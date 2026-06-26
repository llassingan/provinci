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

	"vps-store/internal/repository"
	"vps-store/internal/sse"
)

type NetworkService struct {
	settingsRepo *repository.SettingsRepository
	broker       *sse.EventBroker
	terraformDir string
}

func NewNetworkService(settingsRepo *repository.SettingsRepository, broker *sse.EventBroker, terraformDir string) *NetworkService {
	return &NetworkService{
		settingsRepo: settingsRepo,
		broker:       broker,
		terraformDir: terraformDir,
	}
}

func (s *NetworkService) SetupNetwork(ctx context.Context) error {
	networkChannel := "network"

	defer func() {
		if keyPath := filepath.Join(os.TempDir(), "oci-key.pem"); fileExists(keyPath) {
			os.Remove(keyPath)
		}
	}()

	emitStatus := func(step, message string) {
		s.broker.Publish(networkChannel, sse.SSEEvent{
			Type:    "status",
			Status:  "provisioning",
			Step:    step,
			Message: message,
		})
	}

	emitError := func(message string) {
		s.broker.Publish(networkChannel, sse.SSEEvent{
			Type:    "error",
			Message: message,
		})
	}

	emitStatus("validating_credentials", "Loading OCI settings")

	settings, err := s.settingsRepo.Get(ctx)
	if err != nil {
		emitError("Failed to load settings: " + err.Error())
		return fmt.Errorf("load settings: %w", err)
	}

	if settings.TenancyOCID == "" || settings.UserOCID == "" || settings.Fingerprint == "" ||
		settings.PrivateKey == "" || settings.Region == "" || settings.CompartmentOCID == "" {
		emitError("Missing OCI credentials in settings")
		return fmt.Errorf("incomplete OCI settings")
	}

	keyPath := filepath.Join(os.TempDir(), "oci-key.pem")
	if err := os.WriteFile(keyPath, []byte(settings.PrivateKey), 0600); err != nil {
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
`, settings.Region, settings.CompartmentOCID, settings.TenancyOCID,
		settings.UserOCID, settings.Fingerprint, keyPath)

	if err := os.WriteFile(tfvarsPath, []byte(tfvarsContent), 0644); err != nil {
		emitError("Failed to write terraform.tfvars: " + err.Error())
		return fmt.Errorf("write tfvars: %w", err)
	}

	steps := []struct {
		step    string
		label   string
		cmd     string
		args    []string
	}{
		{"initializing_terraform", "Running terraform init", "terraform", []string{"init"}},
		{"planning_infrastructure", "Running terraform plan", "terraform", []string{"plan"}},
		{"applying_infrastructure", "Running terraform apply", "terraform", []string{"apply", "-auto-approve"}},
	}

	for _, st := range steps {
		emitStatus(st.step, st.label)
		if err := s.runTerraformCmd(st.cmd, st.args, networkChannel); err != nil {
			emitError(fmt.Sprintf("%s failed: %s", st.label, err.Error()))
			return fmt.Errorf("%s: %w", st.step, err)
		}
	}

	emitStatus("parsing_outputs", "Parsing terraform outputs")
	vcnOCID, subnetOCID, err := s.parseOutputs()
	if err != nil {
		emitError("Failed to parse terraform outputs: " + err.Error())
		return fmt.Errorf("parse outputs: %w", err)
	}

	settings.VCNOCID = vcnOCID
	settings.SubnetOCID = subnetOCID
	settings.NetworkProvisioned = true

	if err := s.settingsRepo.Update(ctx, settings); err != nil {
		emitError("Failed to update network settings: " + err.Error())
		return fmt.Errorf("update settings: %w", err)
	}

	emitStatus("ready", "Network setup complete")
	s.broker.Publish(networkChannel, sse.SSEEvent{
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
