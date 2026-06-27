package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"vps-store/internal/model"
	"vps-store/internal/repository"
	"vps-store/internal/service"
)

type VPSHandler struct {
	vpsRepo          *repository.VPSRepository
	tmplRepo         *repository.TemplateRepository
	networkRepo      *repository.NetworkRepository
	settingsRepo     *repository.SettingsRepository
	provisionService *service.VPSProvisionService
}

func NewVPSHandler(
	vpsRepo *repository.VPSRepository,
	tmplRepo *repository.TemplateRepository,
	networkRepo *repository.NetworkRepository,
	settingsRepo *repository.SettingsRepository,
	provisionService *service.VPSProvisionService,
) *VPSHandler {
	return &VPSHandler{
		vpsRepo:          vpsRepo,
		tmplRepo:         tmplRepo,
		networkRepo:      networkRepo,
		settingsRepo:     settingsRepo,
		provisionService: provisionService,
	}
}

type createVPSRequest struct {
	TemplateID         int64    `json:"template_id"`
	NetworkID          int64    `json:"network_id"`
	DisplayName        string   `json:"display_name"`
	Shape              *string  `json:"shape,omitempty"`
	OCPU               *float64 `json:"ocpu,omitempty"`
	MemoryGB           *float64 `json:"memory_gb,omitempty"`
	BootVolumeSizeGB   *int     `json:"boot_volume_size_gb,omitempty"`
}

func (h *VPSHandler) HandleCreateVPS(w http.ResponseWriter, r *http.Request) {
	var req createVPSRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.DisplayName == "" {
		writeError(w, http.StatusBadRequest, "display_name is required")
		return
	}

	if req.NetworkID == 0 {
		writeError(w, http.StatusBadRequest, "network_id is required")
		return
	}

	network, err := h.networkRepo.Get(r.Context(), req.NetworkID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load network")
		return
	}
	if network == nil {
		writeError(w, http.StatusNotFound, "network not found")
		return
	}
	if network.Status != "ready" {
		writeError(w, http.StatusBadRequest, "network is not ready for provisioning")
		return
	}

	template, err := h.tmplRepo.Get(r.Context(), req.TemplateID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load template")
		return
	}
	if template == nil {
		writeError(w, http.StatusNotFound, "template not found")
		return
	}

	vps := &model.VPS{
		DisplayName:      req.DisplayName,
		TemplateID:        req.TemplateID,
		NetworkID:         model.NullInt64{NullInt64: sql.NullInt64{Int64: req.NetworkID, Valid: true}},
		Shape:             template.Shape,
		OCPU:              template.DefaultOCPU,
		MemoryGB:          template.DefaultMemory,
		BootVolumeSizeGB:  template.BootVolumeSizeGB,
		Status:            "pending",
	}

	if req.Shape != nil {
		vps.Shape = *req.Shape
	}
	if req.OCPU != nil {
		vps.OCPU = *req.OCPU
	}
	if req.MemoryGB != nil {
		vps.MemoryGB = *req.MemoryGB
	}
	if req.BootVolumeSizeGB != nil {
		vps.BootVolumeSizeGB = *req.BootVolumeSizeGB
	}

	created, err := h.vpsRepo.Create(r.Context(), vps)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create VPS")
		return
	}

	writeJSON(w, http.StatusOK, created)
}

func (h *VPSHandler) HandleListVPS(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	vpsList, err := h.vpsRepo.List(r.Context(), status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list VPS")
		return
	}

	writeJSON(w, http.StatusOK, vpsList)
}

func (h *VPSHandler) HandleGetVPS(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid VPS id")
		return
	}

	vps, err := h.vpsRepo.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get VPS")
		return
	}
	if vps == nil {
		writeError(w, http.StatusNotFound, "VPS not found")
		return
	}

	writeJSON(w, http.StatusOK, vps)
}

func (h *VPSHandler) HandleDeleteVPS(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid VPS id")
		return
	}

	vps, err := h.vpsRepo.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get VPS")
		return
	}
	if vps == nil {
		writeError(w, http.StatusNotFound, "VPS not found")
		return
	}

	if err := h.vpsRepo.UpdateStatus(r.Context(), id, "terminated"); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete VPS")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *VPSHandler) HandleCredentialsCallback(w http.ResponseWriter, r *http.Request) {
	_, ok := UserIDFromContext(r.Context())
	if !ok {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid VPS id")
			return
		}

		var creds map[string]any
		if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
			writeError(w, http.StatusBadRequest, "invalid credentials body")
			return
		}

		credsJSON, err := json.Marshal(creds)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to marshal credentials")
			return
		}

		if err := h.vpsRepo.UpdateCredentials(r.Context(), id, string(credsJSON)); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update credentials")
			return
		}

		w.WriteHeader(http.StatusNoContent)
		return
	}

	writeError(w, http.StatusForbidden, "forbidden")
}

func (h *VPSHandler) HandleStartVPS(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid VPS id")
		return
	}

	vps, err := h.vpsRepo.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get VPS")
		return
	}
	if vps == nil {
		writeError(w, http.StatusNotFound, "VPS not found")
		return
	}

	if h.provisionService == nil {
		writeError(w, http.StatusServiceUnavailable, "provisioning service not available")
		return
	}

	if err := h.provisionService.StartInstance(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	vps, err = h.vpsRepo.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get updated VPS")
		return
	}

	writeJSON(w, http.StatusOK, vps)
}

func (h *VPSHandler) HandleStopVPS(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid VPS id")
		return
	}

	vps, err := h.vpsRepo.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get VPS")
		return
	}
	if vps == nil {
		writeError(w, http.StatusNotFound, "VPS not found")
		return
	}

	if h.provisionService == nil {
		writeError(w, http.StatusServiceUnavailable, "provisioning service not available")
		return
	}

	if err := h.provisionService.StopInstance(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	vps, err = h.vpsRepo.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get updated VPS")
		return
	}

	writeJSON(w, http.StatusOK, vps)
}

func (h *VPSHandler) HandleRestartVPS(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid VPS id")
		return
	}

	vps, err := h.vpsRepo.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get VPS")
		return
	}
	if vps == nil {
		writeError(w, http.StatusNotFound, "VPS not found")
		return
	}

	if h.provisionService == nil {
		writeError(w, http.StatusServiceUnavailable, "provisioning service not available")
		return
	}

	if err := h.provisionService.RestartInstance(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	vps, err = h.vpsRepo.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get updated VPS")
		return
	}

	writeJSON(w, http.StatusOK, vps)
}

func (h *VPSHandler) HandleResetVPS(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid VPS id")
		return
	}

	vps, err := h.vpsRepo.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get VPS")
		return
	}
	if vps == nil {
		writeError(w, http.StatusNotFound, "VPS not found")
		return
	}

	if h.provisionService == nil {
		writeError(w, http.StatusServiceUnavailable, "provisioning service not available")
		return
	}

	if err := h.provisionService.ResetInstance(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	vps, err = h.vpsRepo.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get updated VPS")
		return
	}

	writeJSON(w, http.StatusOK, vps)
}

type resetPasswordRequest struct {
	Password string `json:"password"`
}

func (h *VPSHandler) HandleResetPasswordVPS(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid VPS id")
		return
	}

	var req resetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Password == "" {
		writeError(w, http.StatusBadRequest, "password is required")
		return
	}

	vps, err := h.vpsRepo.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get VPS")
		return
	}
	if vps == nil {
		writeError(w, http.StatusNotFound, "VPS not found")
		return
	}

	if h.provisionService == nil {
		writeError(w, http.StatusServiceUnavailable, "provisioning service not available")
		return
	}

	if err := h.provisionService.ResetPassword(r.Context(), id, req.Password); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	vps, err = h.vpsRepo.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get updated VPS")
		return
	}

	writeJSON(w, http.StatusOK, vps)
}

func (h *VPSHandler) HandleGetFirewall(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid VPS id")
		return
	}

	_, err = h.vpsRepo.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get VPS")
		return
	}

	if h.provisionService == nil {
		writeError(w, http.StatusServiceUnavailable, "provisioning service not available")
		return
	}

	rules, err := h.provisionService.GetFirewallRules(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"rules": rules})
}

func (h *VPSHandler) HandleUpdateFirewall(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid VPS id")
		return
	}

	var req struct {
		Rules []service.FirewallRule `json:"rules"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	_, err = h.vpsRepo.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get VPS")
		return
	}

	if h.provisionService == nil {
		writeError(w, http.StatusServiceUnavailable, "provisioning service not available")
		return
	}

	if err := h.provisionService.UpdateFirewallRules(r.Context(), id, req.Rules); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "firewall updated"})
}
