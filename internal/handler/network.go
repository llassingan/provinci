package handler

import (
	"net/http"

	"vps-store/internal/repository"
	"vps-store/internal/service"
	"vps-store/internal/sse"
)

type NetworkHandler struct {
	service      *service.NetworkService
	settingsRepo *repository.SettingsRepository
	sseHandler   *SSEHandler
}

func NewNetworkHandler(service *service.NetworkService, settingsRepo *repository.SettingsRepository, sseHandler *SSEHandler) *NetworkHandler {
	return &NetworkHandler{
		service:      service,
		settingsRepo: settingsRepo,
		sseHandler:   sseHandler,
	}
}

func (h *NetworkHandler) HandleNetworkSetup(w http.ResponseWriter, r *http.Request) {
	go func() {
		_ = h.service.SetupNetwork(r.Context())
	}()
	w.WriteHeader(http.StatusAccepted)
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "network_setup_started"})
}

type networkStatusResponse struct {
	Provisioned bool   `json:"provisioned"`
	VCNOCID     string `json:"vcn_ocid"`
	SubnetOCID  string `json:"subnet_ocid"`
}

func (h *NetworkHandler) HandleNetworkStatus(w http.ResponseWriter, r *http.Request) {
	s, err := h.settingsRepo.Get(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load settings")
		return
	}

	resp := networkStatusResponse{
		Provisioned: s.NetworkProvisioned,
		VCNOCID:     s.VCNOCID,
		SubnetOCID:  s.SubnetOCID,
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *NetworkHandler) HandleNetworkEvents(w http.ResponseWriter, r *http.Request) {
	h.sseHandler.HandleNetworkEvents(w, r)
}

func publishNetworkSSE(broker *sse.EventBroker, event sse.SSEEvent) {
	broker.Publish("network", event)
}
