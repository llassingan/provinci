package service

import (
	"context"
	"fmt"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/computeinstanceagent"
	"github.com/oracle/oci-go-sdk/v65/core"

	"vps-store/internal/repository"
	"vps-store/internal/sse"
)

type VPSProvisionService struct {
	computeService  *OCIComputeService
	vpsRepo         *repository.VPSRepository
	networkRepo     *repository.NetworkRepository
	templateRepo    *repository.TemplateRepository
	broker          *sse.EventBroker
	settingsRepo    *repository.SettingsRepository
}

func NewVPSProvisionService(
	computeService *OCIComputeService,
	vpsRepo *repository.VPSRepository,
	networkRepo *repository.NetworkRepository,
	templateRepo *repository.TemplateRepository,
	broker *sse.EventBroker,
	settingsRepo *repository.SettingsRepository,
) *VPSProvisionService {
	return &VPSProvisionService{
		computeService: computeService,
		vpsRepo:        vpsRepo,
		networkRepo:    networkRepo,
		templateRepo:   templateRepo,
		broker:         broker,
		settingsRepo:   settingsRepo,
	}
}

func (s *VPSProvisionService) vpsRegion(ctx context.Context, vpsID int64) (string, error) {
	vps, err := s.vpsRepo.Get(ctx, vpsID)
	if err != nil {
		return "", fmt.Errorf("get vps: %w", err)
	}
	if vps == nil {
		return "", fmt.Errorf("vps %d not found", vpsID)
	}
	if !vps.NetworkID.Valid {
		return "", fmt.Errorf("vps has no network assigned")
	}
	network, err := s.networkRepo.Get(ctx, vps.NetworkID.Int64)
	if err != nil {
		return "", fmt.Errorf("get network: %w", err)
	}
	if network == nil {
		return "", fmt.Errorf("network not found")
	}
	if network.Region == "" {
		return "", fmt.Errorf("network has no region configured")
	}
	return network.Region, nil
}

func (s *VPSProvisionService) StartInstance(ctx context.Context, vpsID int64) error {
	vps, err := s.vpsRepo.Get(ctx, vpsID)
	if err != nil {
		return fmt.Errorf("get vps: %w", err)
	}
	if vps == nil {
		return fmt.Errorf("vps %d not found", vpsID)
	}

	if !vps.OCIInstanceID.Valid || vps.OCIInstanceID.String == "" {
		return fmt.Errorf("vps has no OCI instance ID")
	}

	if vps.Status != "stopped" {
		return fmt.Errorf("vps must be in stopped state to start, current: %s", vps.Status)
	}

	region, err := s.vpsRegion(ctx, vpsID)
	if err != nil {
		return err
	}

	computeClient, err := s.computeService.GetComputeClient(ctx, region)
	if err != nil {
		return fmt.Errorf("get compute client: %w", err)
	}

	action := core.InstanceActionActionStart
	_, err = computeClient.InstanceAction(ctx, core.InstanceActionRequest{
		InstanceId: common.String(vps.OCIInstanceID.String),
		Action:     action,
	})
	if err != nil {
		return fmt.Errorf("instance action start: %w", err)
	}

	if err := s.vpsRepo.UpdateStatus(ctx, vpsID, "running"); err != nil {
		return fmt.Errorf("update vps status: %w", err)
	}

	s.broker.Publish(fmt.Sprintf("vps:%d", vpsID), sse.SSEEvent{
		Type:    "status",
		Status:  "running",
		Step:    "started",
		Message: "VPS instance started",
	})

	return nil
}

func (s *VPSProvisionService) StopInstance(ctx context.Context, vpsID int64) error {
	vps, err := s.vpsRepo.Get(ctx, vpsID)
	if err != nil {
		return fmt.Errorf("get vps: %w", err)
	}
	if vps == nil {
		return fmt.Errorf("vps %d not found", vpsID)
	}

	if !vps.OCIInstanceID.Valid || vps.OCIInstanceID.String == "" {
		return fmt.Errorf("vps has no OCI instance ID")
	}

	if vps.Status != "running" {
		return fmt.Errorf("vps must be in running state to stop, current: %s", vps.Status)
	}

	region, err := s.vpsRegion(ctx, vpsID)
	if err != nil {
		return err
	}

	computeClient, err := s.computeService.GetComputeClient(ctx, region)
	if err != nil {
		return fmt.Errorf("get compute client: %w", err)
	}

	action := core.InstanceActionActionStop
	_, err = computeClient.InstanceAction(ctx, core.InstanceActionRequest{
		InstanceId: common.String(vps.OCIInstanceID.String),
		Action:     action,
	})
	if err != nil {
		return fmt.Errorf("instance action stop: %w", err)
	}

	if err := s.vpsRepo.UpdateStatus(ctx, vpsID, "stopped"); err != nil {
		return fmt.Errorf("update vps status: %w", err)
	}

	s.broker.Publish(fmt.Sprintf("vps:%d", vpsID), sse.SSEEvent{
		Type:    "status",
		Status:  "stopped",
		Step:    "stopped",
		Message: "VPS instance stopped",
	})

	return nil
}

func (s *VPSProvisionService) RestartInstance(ctx context.Context, vpsID int64) error {
	vps, err := s.vpsRepo.Get(ctx, vpsID)
	if err != nil {
		return fmt.Errorf("get vps: %w", err)
	}
	if vps == nil {
		return fmt.Errorf("vps %d not found", vpsID)
	}

	if !vps.OCIInstanceID.Valid || vps.OCIInstanceID.String == "" {
		return fmt.Errorf("vps has no OCI instance ID")
	}

	if vps.Status != "running" {
		return fmt.Errorf("vps must be in running state to restart, current: %s", vps.Status)
	}

	region, err := s.vpsRegion(ctx, vpsID)
	if err != nil {
		return err
	}

	computeClient, err := s.computeService.GetComputeClient(ctx, region)
	if err != nil {
		return fmt.Errorf("get compute client: %w", err)
	}

	action := core.InstanceActionActionSoftreset
	_, err = computeClient.InstanceAction(ctx, core.InstanceActionRequest{
		InstanceId: common.String(vps.OCIInstanceID.String),
		Action:     action,
	})
	if err != nil {
		return fmt.Errorf("instance action restart: %w", err)
	}

	s.broker.Publish(fmt.Sprintf("vps:%d", vpsID), sse.SSEEvent{
		Type:    "status",
		Status:  "running",
		Step:    "restarted",
		Message: "VPS instance restarted",
	})

	return nil
}

func (s *VPSProvisionService) ResetInstance(ctx context.Context, vpsID int64) error {
	vps, err := s.vpsRepo.Get(ctx, vpsID)
	if err != nil {
		return fmt.Errorf("get vps: %w", err)
	}
	if vps == nil {
		return fmt.Errorf("vps %d not found", vpsID)
	}

	if !vps.OCIInstanceID.Valid || vps.OCIInstanceID.String == "" {
		return fmt.Errorf("vps has no OCI instance ID")
	}

	if vps.Status != "running" && vps.Status != "stopped" {
		return fmt.Errorf("vps must be in running or stopped state to reset, current: %s", vps.Status)
	}

	region, err := s.vpsRegion(ctx, vpsID)
	if err != nil {
		return err
	}

	computeClient, err := s.computeService.GetComputeClient(ctx, region)
	if err != nil {
		return fmt.Errorf("get compute client: %w", err)
	}

	action := core.InstanceActionActionReset
	_, err = computeClient.InstanceAction(ctx, core.InstanceActionRequest{
		InstanceId: common.String(vps.OCIInstanceID.String),
		Action:     action,
	})
	if err != nil {
		return fmt.Errorf("instance action reset: %w", err)
	}

	s.broker.Publish(fmt.Sprintf("vps:%d", vpsID), sse.SSEEvent{
		Type:    "status",
		Status:  "running",
		Step:    "reset",
		Message: "VPS instance reset",
	})

	return nil
}

func (s *VPSProvisionService) ResetPassword(ctx context.Context, vpsID int64, newPassword string) error {
	vps, err := s.vpsRepo.Get(ctx, vpsID)
	if err != nil {
		return fmt.Errorf("get vps: %w", err)
	}
	if vps == nil {
		return fmt.Errorf("vps %d not found", vpsID)
	}

	if !vps.OCIInstanceID.Valid || vps.OCIInstanceID.String == "" {
		return fmt.Errorf("vps has no OCI instance ID")
	}

	if vps.Status != "running" {
		return fmt.Errorf("vps must be running to reset password, current: %s", vps.Status)
	}

	region, err := s.vpsRegion(ctx, vpsID)
	if err != nil {
		return err
	}

	instanceAgentClient, err := s.computeService.GetInstanceAgentClient(ctx, region)
	if err != nil {
		return fmt.Errorf("get instance agent client: %w", err)
	}

	compartmentOCID, err := s.computeService.GetCompartmentOCID(ctx)
	if err != nil {
		return fmt.Errorf("get compartment ocid: %w", err)
	}

	commandText := fmt.Sprintf("echo 'root:%s' | chpasswd", shellEscape(newPassword))

	timeoutSeconds := 30

	_, err = instanceAgentClient.CreateInstanceAgentCommand(ctx, computeinstanceagent.CreateInstanceAgentCommandRequest{
		CreateInstanceAgentCommandDetails: computeinstanceagent.CreateInstanceAgentCommandDetails{
			CompartmentId: common.String(compartmentOCID),
			Target: &computeinstanceagent.InstanceAgentCommandTarget{
				InstanceId: common.String(vps.OCIInstanceID.String),
			},
			Content: &computeinstanceagent.InstanceAgentCommandContent{
				Source: computeinstanceagent.InstanceAgentCommandSourceViaTextDetails{
					Text:      common.String(commandText),
					TextSha256: nil,
				},
				Output: &computeinstanceagent.InstanceAgentCommandOutputViaTextDetails{},
			},
			ExecutionTimeOutInSeconds: &timeoutSeconds,
		},
	})
	if err != nil {
		return fmt.Errorf("create instance agent command: %w", err)
	}

	s.broker.Publish(fmt.Sprintf("vps:%d", vpsID), sse.SSEEvent{
		Type:    "status",
		Status:  "running",
		Step:    "password_reset",
		Message: "Password reset command sent to VPS instance",
	})

	return nil
}

func shellEscape(s string) string {
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\'' {
			result = append(result, '\'', '\\', '\'', '\'')
		} else {
			result = append(result, c)
		}
	}
	return string(result)
}

type FirewallRule struct {
	Port        int    `json:"port"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Direction   string `json:"direction"`
	Source      string `json:"source,omitempty"`
	Destination string `json:"destination,omitempty"`
}

func (s *VPSProvisionService) GetFirewallRules(ctx context.Context, vpsID int64) ([]FirewallRule, error) {
	vps, err := s.vpsRepo.Get(ctx, vpsID)
	if err != nil {
		return nil, fmt.Errorf("get vps: %w", err)
	}
	if vps == nil {
		return nil, fmt.Errorf("vps %d not found", vpsID)
	}

	if !vps.NetworkID.Valid {
		return nil, fmt.Errorf("vps has no network assigned")
	}

	network, err := s.networkRepo.Get(ctx, vps.NetworkID.Int64)
	if err != nil {
		return nil, fmt.Errorf("get network: %w", err)
	}
	if network == nil {
		return nil, fmt.Errorf("network not found")
	}

	networkClient, err := s.computeService.GetNetworkClient(ctx, network.Region)
	if err != nil {
		return nil, fmt.Errorf("get network client: %w", err)
	}

	compartmentOCID, err := s.computeService.GetCompartmentOCID(ctx)
	if err != nil {
		return nil, fmt.Errorf("get compartment ocid: %w", err)
	}

	listResp, err := networkClient.ListSecurityLists(ctx, core.ListSecurityListsRequest{
		CompartmentId: common.String(compartmentOCID),
		VcnId:         common.String(network.VCNOCID),
	})
	if err != nil {
		return nil, fmt.Errorf("list security lists: %w", err)
	}

	if len(listResp.Items) == 0 {
		return nil, fmt.Errorf("no security lists found for VCN %s", network.VCNOCID)
	}

	securityListID := listResp.Items[0].Id

	getResp, err := networkClient.GetSecurityList(ctx, core.GetSecurityListRequest{
		SecurityListId: securityListID,
	})
	if err != nil {
		return nil, fmt.Errorf("get security list: %w", err)
	}

	var rules []FirewallRule

	for _, r := range getResp.IngressSecurityRules {
		rule := FirewallRule{
			Direction: "ingress",
		}
		if r.Protocol != nil {
			if *r.Protocol == "6" || *r.Protocol == "17" {
				if r.TcpOptions != nil && r.TcpOptions.DestinationPortRange != nil {
					rule.Port = *r.TcpOptions.DestinationPortRange.Min
				}
			}
		}
		if r.Source != nil {
			rule.Source = *r.Source
		}
		if r.Description != nil {
			rule.Description = *r.Description
		}
		rules = append(rules, rule)
	}

	for _, r := range getResp.EgressSecurityRules {
		rule := FirewallRule{
			Direction: "egress",
		}
		if r.Protocol != nil {
			if *r.Protocol == "6" || *r.Protocol == "17" {
				if r.TcpOptions != nil && r.TcpOptions.DestinationPortRange != nil {
					rule.Port = *r.TcpOptions.DestinationPortRange.Min
				}
			}
		}
		if r.Destination != nil {
			rule.Destination = *r.Destination
		}
		if r.Description != nil {
			rule.Description = *r.Description
		}
		rules = append(rules, rule)
	}

	if rules == nil {
		rules = []FirewallRule{}
	}
	return rules, nil
}

func (s *VPSProvisionService) UpdateFirewallRules(ctx context.Context, vpsID int64, rules []FirewallRule) error {
	vps, err := s.vpsRepo.Get(ctx, vpsID)
	if err != nil {
		return fmt.Errorf("get vps: %w", err)
	}
	if vps == nil {
		return fmt.Errorf("vps %d not found", vpsID)
	}

	if !vps.NetworkID.Valid {
		return fmt.Errorf("vps has no network assigned")
	}

	network, err := s.networkRepo.Get(ctx, vps.NetworkID.Int64)
	if err != nil {
		return fmt.Errorf("get network: %w", err)
	}
	if network == nil {
		return fmt.Errorf("network not found")
	}

	networkClient, err := s.computeService.GetNetworkClient(ctx, network.Region)
	if err != nil {
		return fmt.Errorf("get network client: %w", err)
	}

	compartmentOCID, err := s.computeService.GetCompartmentOCID(ctx)
	if err != nil {
		return fmt.Errorf("get compartment ocid: %w", err)
	}

	listResp, err := networkClient.ListSecurityLists(ctx, core.ListSecurityListsRequest{
		CompartmentId: common.String(compartmentOCID),
		VcnId:         common.String(network.VCNOCID),
	})
	if err != nil {
		return fmt.Errorf("list security lists: %w", err)
	}

	if len(listResp.Items) == 0 {
		return fmt.Errorf("no security lists found for VCN %s", network.VCNOCID)
	}

	securityListID := listResp.Items[0].Id

	var ingressRules []core.IngressSecurityRule
	var egressRules []core.EgressSecurityRule

	for _, r := range rules {
		portMin := r.Port
		portMax := r.Port
		if portMin == 0 {
			portMin = 80
			portMax = 80
		}

		portRange := core.PortRange{
			Min: &portMin,
			Max: &portMax,
		}

		protocol := common.String("6")

		if r.Direction == "ingress" {
			ingressRules = append(ingressRules, core.IngressSecurityRule{
				Protocol:    protocol,
				Source:      common.String(r.Source),
				TcpOptions:  &core.TcpOptions{DestinationPortRange: &portRange},
			})
		} else if r.Direction == "egress" {
			egressRules = append(egressRules, core.EgressSecurityRule{
				Protocol:    protocol,
				Destination: common.String(r.Destination),
				TcpOptions:  &core.TcpOptions{DestinationPortRange: &portRange},
			})
		}
	}

	egressRules = append(egressRules, core.EgressSecurityRule{
		Protocol:    common.String("all"),
		Destination: common.String("0.0.0.0/0"),
	})

	_, err = networkClient.UpdateSecurityList(ctx, core.UpdateSecurityListRequest{
		SecurityListId: securityListID,
		UpdateSecurityListDetails: core.UpdateSecurityListDetails{
			IngressSecurityRules: ingressRules,
			EgressSecurityRules:  egressRules,
		},
	})
	if err != nil {
		return fmt.Errorf("update security list: %w", err)
	}

	return nil
}
