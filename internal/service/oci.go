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

	mu              sync.Mutex
	initialized     bool
	compartmentOCID string
	tenancyOCID     string
	userOCID        string
	fingerprint     string
	privateKey      string
	cachedClients   map[string]*regionClients
}

type regionClients struct {
	computeClient        core.ComputeClient
	virtualNetworkClient core.VirtualNetworkClient
	instanceAgentClient  computeinstanceagent.ComputeInstanceAgentClient
}

func NewOCIComputeService(settingsRepo *repository.SettingsRepository) *OCIComputeService {
	return &OCIComputeService{
		settingsRepo:  settingsRepo,
		cachedClients: make(map[string]*regionClients),
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
		settings.PrivateKey == "" || settings.CompartmentOCID == "" {
		return fmt.Errorf("incomplete OCI settings")
	}

	s.tenancyOCID = settings.TenancyOCID
	s.userOCID = settings.UserOCID
	s.fingerprint = settings.Fingerprint
	s.privateKey = settings.PrivateKey
	s.compartmentOCID = settings.CompartmentOCID
	s.initialized = true
	return nil
}

func (s *OCIComputeService) clientProvider(region string) common.ConfigurationProvider {
	return common.NewRawConfigurationProvider(
		s.tenancyOCID,
		s.userOCID,
		region,
		s.fingerprint,
		s.privateKey,
		nil,
	)
}

func (s *OCIComputeService) getOrCreateClients(ctx context.Context, region string) (*regionClients, error) {
	if err := s.init(ctx); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if c, ok := s.cachedClients[region]; ok {
		return c, nil
	}

	provider := s.clientProvider(region)

	computeClient, err := core.NewComputeClientWithConfigurationProvider(provider)
	if err != nil {
		return nil, fmt.Errorf("create compute client for %s: %w", region, err)
	}

	virtualNetworkClient, err := core.NewVirtualNetworkClientWithConfigurationProvider(provider)
	if err != nil {
		return nil, fmt.Errorf("create virtual network client for %s: %w", region, err)
	}

	instanceAgentClient, err := computeinstanceagent.NewComputeInstanceAgentClientWithConfigurationProvider(provider)
	if err != nil {
		return nil, fmt.Errorf("create instance agent client for %s: %w", region, err)
	}

	c := &regionClients{
		computeClient:        computeClient,
		virtualNetworkClient: virtualNetworkClient,
		instanceAgentClient:  instanceAgentClient,
	}
	s.cachedClients[region] = c
	return c, nil
}

func (s *OCIComputeService) GetComputeClient(ctx context.Context, region string) (core.ComputeClient, error) {
	c, err := s.getOrCreateClients(ctx, region)
	if err != nil {
		return core.ComputeClient{}, err
	}
	return c.computeClient, nil
}

func (s *OCIComputeService) GetNetworkClient(ctx context.Context, region string) (core.VirtualNetworkClient, error) {
	c, err := s.getOrCreateClients(ctx, region)
	if err != nil {
		return core.VirtualNetworkClient{}, err
	}
	return c.virtualNetworkClient, nil
}

func (s *OCIComputeService) GetInstanceAgentClient(ctx context.Context, region string) (computeinstanceagent.ComputeInstanceAgentClient, error) {
	c, err := s.getOrCreateClients(ctx, region)
	if err != nil {
		return computeinstanceagent.ComputeInstanceAgentClient{}, err
	}
	return c.instanceAgentClient, nil
}

func (s *OCIComputeService) GetCompartmentOCID(ctx context.Context) (string, error) {
	if err := s.init(ctx); err != nil {
		return "", err
	}
	return s.compartmentOCID, nil
}
