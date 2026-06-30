package service

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"vps-store/internal/logger"
	"vps-store/internal/repository"
	"vps-store/internal/sse"
)

type NetworkService struct {
	settingsRepo *repository.SettingsRepository
	networkRepo  *repository.NetworkRepository
	broker       *sse.EventBroker
	terraformDir string
	log          *logger.Logger
}

func NewNetworkService(
	settingsRepo *repository.SettingsRepository,
	networkRepo *repository.NetworkRepository,
	broker *sse.EventBroker,
	terraformDir string,
	log *logger.Logger,
) *NetworkService {
	return &NetworkService{
		settingsRepo: settingsRepo,
		networkRepo:  networkRepo,
		broker:       broker,
		terraformDir: terraformDir,
		log:          log,
	}
}

func (s *NetworkService) ProvisionNetwork(ctx context.Context, networkID int64) error {
	channel := fmt.Sprintf("network:%d", networkID)

	s.log.Debug("provision_network_start", "network_id", networkID, "channel", channel)

	defer func() {
		if keyPath := filepath.Join(os.TempDir(), "oci-key.pem"); fileExists(keyPath) {
			os.Remove(keyPath)
			s.log.Debug("cleaned_up_temp_key", "path", keyPath)
		}
	}()

	emitStatus := func(step, message string) {
		s.broker.Publish(channel, sse.SSEEvent{
			Type:    "status",
			Status:  "provisioning",
			Step:    step,
			Message: message,
		})
	}

	emitError := func(message string) {
		s.broker.Publish(channel, sse.SSEEvent{
			Type:    "error",
			Message: message,
		})
	}

	network, err := s.networkRepo.Get(ctx, networkID)
	if err != nil || network == nil {
		if network == nil {
			s.log.Error("network_not_found", "network_id", networkID)
			emitError("Network not found")
			return fmt.Errorf("network %d not found", networkID)
		}
		s.log.Error("get_network_failed", "network_id", networkID, "error", err)
		emitError("Failed to load network: " + err.Error())
		return fmt.Errorf("get network: %w", err)
	}

	s.log.Debug("network_loaded", "network_id", networkID, "name", network.Name, "region", network.Region, "status", network.Status, "cidr_vcn", network.CIDRVCN, "cidr_subnet", network.CIDRSubnet)

	if network.Status == "provisioning" {
		emitError("Network is already provisioning")
		return fmt.Errorf("network %d is already provisioning", networkID)
	}

	if err := s.networkRepo.UpdateStatus(ctx, networkID, "provisioning"); err != nil {
		emitError("Failed to update network status: " + err.Error())
		return fmt.Errorf("update status: %w", err)
	}

	emitStatus("loading_credentials", "Loading OCI settings")

	s.log.Debug("loading_oci_settings")
	settings, err := s.settingsRepo.Get(ctx)
	if err != nil {
		s.log.Error("load_settings_failed", "error", err)
		s.networkRepo.UpdateStatus(ctx, networkID, "failed")
		emitError("Failed to load settings: " + err.Error())
		return fmt.Errorf("load settings: %w", err)
	}

	if settings.TenancyOCID == "" || settings.UserOCID == "" || settings.Fingerprint == "" ||
		settings.PrivateKey == "" || settings.Region == "" || settings.CompartmentOCID == "" {
		s.log.Error("incomplete_oci_settings", "has_tenancy", settings.TenancyOCID != "", "has_user", settings.UserOCID != "", "has_fingerprint", settings.Fingerprint != "", "has_private_key", settings.PrivateKey != "", "has_region", settings.Region != "", "has_compartment", settings.CompartmentOCID != "")
		s.networkRepo.UpdateStatus(ctx, networkID, "failed")
		emitError("Missing OCI credentials in settings")
		return fmt.Errorf("incomplete OCI settings")
	}

	s.log.Debug("oci_settings_loaded", "region", settings.Region, "compartment_ocid", maskOCID(settings.CompartmentOCID), "tenancy_ocid", maskOCID(settings.TenancyOCID))

	keyPath := filepath.Join(os.TempDir(), "oci-key.pem")
	if err := os.WriteFile(keyPath, []byte(settings.PrivateKey), 0600); err != nil {
		s.networkRepo.UpdateStatus(ctx, networkID, "failed")
		emitError("Failed to write private key: " + err.Error())
		return fmt.Errorf("write key: %w", err)
	}

	tfvarsPath := filepath.Join(s.terraformDir, "terraform.tfvars")
	tfvarsContent := fmt.Sprintf(`region           = %q
compartment_ocid = %q
tenancy_ocid     = %q
user_ocid        = %q
fingerprint      = %q
private_key_path = %q
vcn_cidr_block   = %q
subnet_cidr_block = %q
display_name     = %q
dns_label        = %q
`, network.Region, settings.CompartmentOCID, settings.TenancyOCID,
		settings.UserOCID, settings.Fingerprint, keyPath,
		network.CIDRVCN, network.CIDRSubnet, network.Name, safeDNSLabel(network.Name))

	if err := os.WriteFile(tfvarsPath, []byte(tfvarsContent), 0644); err != nil {
		s.log.Error("write_tfvars_failed", "path", tfvarsPath, "error", err)
		s.networkRepo.UpdateStatus(ctx, networkID, "failed")
		emitError("Failed to write terraform.tfvars: " + err.Error())
		return fmt.Errorf("write tfvars: %w", err)
	}

	s.log.Debug("tfvars_written", "path", tfvarsPath, "region", network.Region)

	steps := []struct {
		step  string
		label string
		cmd   string
		args  []string
	}{
		{"initializing_terraform", "Running terraform init", "terraform", []string{"init"}},
		{"planning_infrastructure", "Running terraform plan", "terraform", []string{"plan"}},
		{"applying_infrastructure", "Running terraform apply", "terraform", []string{"apply", "-auto-approve"}},
	}

	for _, st := range steps {
		emitStatus(st.step, st.label)
		s.log.Debug("running_terraform_step", "step", st.step, "cmd", st.cmd, "args", st.args)
		if err := s.runTerraformCmd(st.cmd, st.args, channel); err != nil {
			s.log.Error("terraform_step_failed", "step", st.step, "error", err)
			s.networkRepo.UpdateStatus(ctx, networkID, "failed")
			emitError(fmt.Sprintf("%s failed: %s", st.label, err.Error()))
			return fmt.Errorf("%s: %w", st.step, err)
		}
		s.log.Debug("terraform_step_complete", "step", st.step)
	}

	emitStatus("parsing_outputs", "Parsing terraform outputs")
	s.log.Debug("parsing_terraform_outputs")
	vcnOCID, subnetOCID, err := s.parseOutputs()
	if err != nil {
		s.log.Error("parse_outputs_failed", "error", err)
		s.networkRepo.UpdateStatus(ctx, networkID, "failed")
		emitError("Failed to parse terraform outputs: " + err.Error())
		return fmt.Errorf("parse outputs: %w", err)
	}
	s.log.Debug("terraform_outputs_parsed", "vcn_ocid", vcnOCID, "subnet_ocid", subnetOCID)

	if err := s.networkRepo.UpdateProvisionResult(ctx, networkID, vcnOCID, subnetOCID); err != nil {
		emitError("Failed to update network: " + err.Error())
		return fmt.Errorf("update network: %w", err)
	}

	emitStatus("ready", "Network setup complete")
	s.broker.Publish(channel, sse.SSEEvent{
		Type:    "status",
		Status:  "ready",
		Step:    "ready",
		Message: "Network infrastructure provisioned",
		Data: map[string]string{
			"vcn_ocid":    vcnOCID,
			"subnet_ocid": subnetOCID,
		},
	})

	return nil
}

func (s *NetworkService) DestroyNetwork(ctx context.Context, networkID int64) error {
	channel := fmt.Sprintf("network:%d", networkID)

	s.log.Debug("destroy_network_start", "network_id", networkID)

	network, err := s.networkRepo.Get(ctx, networkID)
	if err != nil || network == nil {
		if network == nil {
			return fmt.Errorf("network %d not found", networkID)
		}
		return fmt.Errorf("get network: %w", err)
	}

	settings, err := s.settingsRepo.Get(ctx)
	if err != nil {
		return fmt.Errorf("load settings: %w", err)
	}

	keyPath := filepath.Join(os.TempDir(), "oci-key.pem")
	if err := os.WriteFile(keyPath, []byte(settings.PrivateKey), 0600); err != nil {
		return fmt.Errorf("write key: %w", err)
	}
	defer func() {
		if fileExists(keyPath) {
			os.Remove(keyPath)
		}
	}()

	tfvarsPath := filepath.Join(s.terraformDir, "terraform.tfvars")
	tfvarsContent := fmt.Sprintf(`region           = %q
compartment_ocid = %q
tenancy_ocid     = %q
user_ocid        = %q
fingerprint      = %q
private_key_path = %q
vcn_cidr_block   = %q
subnet_cidr_block = %q
display_name     = %q
dns_label        = %q
`, network.Region, settings.CompartmentOCID, settings.TenancyOCID,
		settings.UserOCID, settings.Fingerprint, keyPath,
		network.CIDRVCN, network.CIDRSubnet, network.Name, safeDNSLabel(network.Name))

	if err := os.WriteFile(tfvarsPath, []byte(tfvarsContent), 0644); err != nil {
		return fmt.Errorf("write tfvars: %w", err)
	}

	s.log.Debug("destroy_network_terraform", "network_id", networkID, "name", network.Name)
	cmd := exec.Command("terraform", "destroy", "-auto-approve")
	cmd.Dir = s.terraformDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		s.broker.Publish(channel, sse.SSEEvent{
			Type:      "status",
			Status:    "destroying",
			Step:      "terraform_output",
			Message:   line,
			Timestamp: time.Now().UnixMilli(),
		})
	}

	errScanner := bufio.NewScanner(stderr)
	var errOutput []string
	for errScanner.Scan() {
		line := errScanner.Text()
		errOutput = append(errOutput, line)
	}

	if err := cmd.Wait(); err != nil {
		s.log.Error("terraform_destroy_failed", "network_id", networkID, "error", err)
		return fmt.Errorf("terraform destroy: %w\n%s", err, formatOutput(errOutput))
	}

	s.log.Debug("destroy_network_complete", "network_id", networkID)

	if err := s.networkRepo.Delete(ctx, networkID); err != nil {
		s.log.Error("destroy_network_db_delete_failed", "network_id", networkID, "error", err)
		return fmt.Errorf("delete network record: %w", err)
	}

	return nil
}

func (s *NetworkService) runTerraformCmd(command string, args []string, channel string) error {
	cmd := exec.Command(command, args...)
	cmd.Dir = s.terraformDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		s.broker.Publish(channel, sse.SSEEvent{
			Type:      "status",
			Status:    "provisioning",
			Step:      "terraform_output",
			Message:   line,
			Timestamp: time.Now().UnixMilli(),
		})
	}

	errScanner := bufio.NewScanner(stderr)
	var errOutput []string
	for errScanner.Scan() {
		line := errScanner.Text()
		errOutput = append(errOutput, line)
		s.broker.Publish(channel, sse.SSEEvent{
			Type:      "status",
			Status:    "provisioning",
			Step:      "terraform_output",
			Message:   line,
			Timestamp: time.Now().UnixMilli(),
		})
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("terraform %s: %w\n%s", command, err, formatOutput(errOutput))
	}

	return nil
}

func (s *NetworkService) parseOutputs() (string, string, error) {
	cmd := exec.Command("terraform", "output", "-json")
	cmd.Dir = s.terraformDir
	out, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("terraform output: %w", err)
	}

	var outputs map[string]struct {
		Value string `json:"value"`
		Type  string `json:"type"`
	}
	if err := json.Unmarshal(out, &outputs); err != nil {
		return "", "", fmt.Errorf("parse output json: %w", err)
	}

	vcnOCID, ok := outputs["vcn_ocid"]
	if !ok {
		return "", "", fmt.Errorf("missing vcn_ocid in terraform outputs")
	}
	subnetOCID, ok := outputs["subnet_ocid"]
	if !ok {
		return "", "", fmt.Errorf("missing subnet_ocid in terraform outputs")
	}

	return vcnOCID.Value, subnetOCID.Value, nil
}

func safeDNSLabel(name string) string {
	result := make([]byte, 0, len(name))
	for i := 0; i < len(name) && len(result) < 15; i++ {
		c := name[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			result = append(result, c)
		} else if c >= 'A' && c <= 'Z' {
			result = append(result, c+32)
		}
		// skip hyphens, underscores, and all other non-alphanumeric chars —
		// OCI dns_label must be strictly alphanumeric
	}
	if len(result) == 0 {
		return "network"
	}
	if result[0] < 'a' || result[0] > 'z' {
		return "n" + string(result)
	}
	return string(result)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func formatOutput(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	result := ""
	for _, l := range lines {
		result += l + "\n"
	}
	return result
}

func maskOCID(ocid string) string {
	if len(ocid) <= 20 {
		return "***"
	}
	return ocid[:10] + "..." + ocid[len(ocid)-10:]
}
