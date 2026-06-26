package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"vps-store/internal/model"
	"vps-store/internal/repository"
)

type VPSHandler struct {
	vpsRepo    *repository.VPSRepository
	tmplRepo   *repository.TemplateRepository
	settingsRepo *repository.SettingsRepository
}

func NewVPSHandler(
	vpsRepo *repository.VPSRepository,
	tmplRepo *repository.TemplateRepository,
	settingsRepo *repository.SettingsRepository,
) *VPSHandler {
	return &VPSHandler{
		vpsRepo:      vpsRepo,
		tmplRepo:     tmplRepo,
		settingsRepo: settingsRepo,
	}
}

type createVPSRequest struct {
	TemplateID         int64   `json:"template_id"`
	DisplayName        string  `json:"display_name"`
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
		TemplateID:       req.TemplateID,
		Shape:            template.Shape,
		OCPU:             template.DefaultOCPU,
		MemoryGB:         template.DefaultMemory,
		BootVolumeSizeGB: template.BootVolumeSizeGB,
		Status:           "pending",
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
