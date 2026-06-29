package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/computeinstanceagent"
	"github.com/oracle/oci-go-sdk/v65/core"

	"vps-store/internal/logger"
	"vps-store/internal/repository"
)

type OCIComputeService struct {
	settingsRepo *repository.SettingsRepository
	log          *logger.Logger

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

func NewOCIComputeService(settingsRepo *repository.SettingsRepository, log *logger.Logger) *OCIComputeService {
	return &OCIComputeService{
		settingsRepo:  settingsRepo,
		log:           log,
		cachedClients: make(map[string]*regionClients),
	}
}

func (s *OCIComputeService) init(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.initialized {
		return nil
	}

	s.log.Debug("oci_init_start")
	settings, err := s.settingsRepo.Get(ctx)
	if err != nil {
		s.log.Error("oci_init_get_settings_failed", "error", err)
		return fmt.Errorf("get settings: %w", err)
	}
	if settings == nil {
		s.log.Error("oci_init_no_settings")
		return fmt.Errorf("no OCI settings configured")
	}

	if settings.TenancyOCID == "" || settings.UserOCID == "" || settings.Fingerprint == "" ||
		settings.PrivateKey == "" || settings.CompartmentOCID == "" {
		s.log.Error("oci_init_incomplete_settings", "has_tenancy", settings.TenancyOCID != "", "has_user", settings.UserOCID != "", "has_fingerprint", settings.Fingerprint != "", "has_private_key", settings.PrivateKey != "", "has_compartment", settings.CompartmentOCID != "")
		return fmt.Errorf("incomplete OCI settings")
	}

	s.tenancyOCID = settings.TenancyOCID
	s.userOCID = settings.UserOCID
	s.fingerprint = settings.Fingerprint
	s.privateKey = settings.PrivateKey
	s.compartmentOCID = settings.CompartmentOCID
	s.initialized = true
	s.log.Debug("oci_init_complete", "compartment_ocid", maskOCID(s.compartmentOCID), "tenancy_ocid", maskOCID(s.tenancyOCID))
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

	s.log.Debug("creating_oci_clients", "region", region)

	provider := s.clientProvider(region)

	computeClient, err := core.NewComputeClientWithConfigurationProvider(provider)
	if err != nil {
		s.log.Error("create_compute_client_failed", "region", region, "error", err)
		return nil, fmt.Errorf("create compute client for %s: %w", region, err)
	}

	virtualNetworkClient, err := core.NewVirtualNetworkClientWithConfigurationProvider(provider)
	if err != nil {
		s.log.Error("create_network_client_failed", "region", region, "error", err)
		return nil, fmt.Errorf("create virtual network client for %s: %w", region, err)
	}

	instanceAgentClient, err := computeinstanceagent.NewComputeInstanceAgentClientWithConfigurationProvider(provider)
	if err != nil {
		s.log.Error("create_instance_agent_client_failed", "region", region, "error", err)
		return nil, fmt.Errorf("create instance agent client for %s: %w", region, err)
	}

	c := &regionClients{
		computeClient:        computeClient,
		virtualNetworkClient: virtualNetworkClient,
		instanceAgentClient:  instanceAgentClient,
	}
	s.cachedClients[region] = c
	s.log.Debug("oci_clients_created", "region", region)
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
