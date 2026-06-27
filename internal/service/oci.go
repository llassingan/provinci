package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/computeinstanceagent"
	"github.com/oracle/oci-go-sdk/v65/core"

	"vps-store/internal/repository"
)

type OCIComputeService struct {
	settingsRepo *repository.SettingsRepository

	mu                    sync.Mutex
	initialized           bool
	configProvider        common.ConfigurationProvider
	compartmentOCID       string
	computeClient         core.ComputeClient
	virtualNetworkClient  core.VirtualNetworkClient
	instanceAgentClient   computeinstanceagent.ComputeInstanceAgentClient
}

func NewOCIComputeService(settingsRepo *repository.SettingsRepository) *OCIComputeService {
	return &OCIComputeService{
		settingsRepo: settingsRepo,
	}
}

func (s *OCIComputeService) init(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.initialized {
		return nil
	}

	settings, err := s.settingsRepo.Get(ctx)
	if err != nil {
		return fmt.Errorf("get settings: %w", err)
	}
	if settings == nil {
		return fmt.Errorf("no OCI settings configured")
	}

	if settings.TenancyOCID == "" || settings.UserOCID == "" || settings.Fingerprint == "" ||
		settings.PrivateKey == "" || settings.Region == "" || settings.CompartmentOCID == "" {
		return fmt.Errorf("incomplete OCI settings")
	}

	s.configProvider = common.NewRawConfigurationProvider(
		settings.TenancyOCID,
		settings.UserOCID,
		settings.Region,
		settings.Fingerprint,
		settings.PrivateKey,
		nil,
	)
	s.compartmentOCID = settings.CompartmentOCID

	computeClient, err := core.NewComputeClientWithConfigurationProvider(s.configProvider)
	if err != nil {
		return fmt.Errorf("create compute client: %w", err)
	}
	s.computeClient = computeClient

	virtualNetworkClient, err := core.NewVirtualNetworkClientWithConfigurationProvider(s.configProvider)
	if err != nil {
		return fmt.Errorf("create virtual network client: %w", err)
	}
	s.virtualNetworkClient = virtualNetworkClient

	instanceAgentClient, err := computeinstanceagent.NewComputeInstanceAgentClientWithConfigurationProvider(s.configProvider)
	if err != nil {
		return fmt.Errorf("create instance agent client: %w", err)
	}
	s.instanceAgentClient = instanceAgentClient

	s.initialized = true
	return nil
}

func (s *OCIComputeService) GetComputeClient(ctx context.Context) (core.ComputeClient, error) {
	if err := s.init(ctx); err != nil {
		return core.ComputeClient{}, err
	}
	return s.computeClient, nil
}

func (s *OCIComputeService) GetNetworkClient(ctx context.Context) (core.VirtualNetworkClient, error) {
	if err := s.init(ctx); err != nil {
		return core.VirtualNetworkClient{}, err
	}
	return s.virtualNetworkClient, nil
}

func (s *OCIComputeService) GetInstanceAgentClient(ctx context.Context) (computeinstanceagent.ComputeInstanceAgentClient, error) {
	if err := s.init(ctx); err != nil {
		return computeinstanceagent.ComputeInstanceAgentClient{}, err
	}
	return s.instanceAgentClient, nil
}

func (s *OCIComputeService) GetCompartmentOCID(ctx context.Context) (string, error) {
	if err := s.init(ctx); err != nil {
		return "", err
	}
	return s.compartmentOCID, nil
}
