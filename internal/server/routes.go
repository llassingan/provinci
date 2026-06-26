package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"vps-store/internal/handler"
)

func (s *Server) mountRoutes() {
	r := s.router
	authHandler := handler.NewAuthHandler(s.authService)

	r.Get("/api/health", handleHealth)

	r.Post("/api/auth/signup", authHandler.HandleSignup)
	r.Post("/api/auth/login", authHandler.HandleLogin)
	r.Post("/api/auth/logout", authHandler.HandleLogout)

	r.Group(func(r chi.Router) {
		r.Use(AuthMiddleware(s.authService))

		r.Post("/api/vps", s.vpsHandler.HandleCreateVPS)
		r.Get("/api/vps", s.vpsHandler.HandleListVPS)
		r.Get("/api/vps/{id}", s.vpsHandler.HandleGetVPS)
		r.Delete("/api/vps/{id}", s.vpsHandler.HandleDeleteVPS)

		if s.sseHandler != nil {
			r.Get("/api/vps/{id}/events", s.sseHandler.HandleVPSEvents)
			r.Get("/api/network/events", s.sseHandler.HandleNetworkEvents)
		} else {
			r.Get("/api/vps/{id}/events", handleSSEEventsStub)
			r.Get("/api/network/events", handleNetworkSSEStub)
		}

		if s.templateHandler != nil {
			r.Get("/api/templates", s.templateHandler.HandleListTemplates)
			r.Post("/api/templates", s.templateHandler.HandleCreateTemplate)
		} else {
			r.Get("/api/templates", handleListTemplatesStub)
			r.Post("/api/templates", handleCreateTemplateStub)
		}

		if s.settingsHandler != nil {
			r.Get("/api/settings", s.settingsHandler.HandleGetSettings)
			r.Put("/api/settings", s.settingsHandler.HandleUpdateSettings)
		} else {
			r.Get("/api/settings", handleGetSettingsStub)
			r.Put("/api/settings", handleUpdateSettingsStub)
		}

		if s.networkHandler != nil {
			r.Post("/api/network/setup", s.networkHandler.HandleNetworkSetup)
			r.Get("/api/network/status", s.networkHandler.HandleNetworkStatus)
		} else {
			r.Post("/api/network/setup", handleNetworkSetupStub)
			r.Get("/api/network/status", handleNetworkStatusStub)
		}
	})

	r.Post("/api/vps/{id}/credentials", s.vpsHandler.HandleCredentialsCallback)
}

func handleListTemplatesStub(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "not implemented yet"})
}

func handleCreateTemplateStub(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "not implemented yet"})
}

func handleGetSettingsStub(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "not implemented yet"})
}

func handleUpdateSettingsStub(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "not implemented yet"})
}

func handleNetworkSetupStub(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "not implemented yet"})
}

func handleNetworkStatusStub(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "not implemented yet"})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func handleCreateVPSStub(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "not implemented yet"})
}

func handleListVPSStub(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "not implemented yet"})
}

func handleGetVPSStub(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "not implemented yet"})
}

func handleDeleteVPSStub(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "not implemented yet"})
}

func handleSSEEventsStub(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "not implemented yet"})
}

func handleNetworkSSEStub(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "not implemented yet"})
}

func handleCredentialsCallbackStub(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "not implemented yet"})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
