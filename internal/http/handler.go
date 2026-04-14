package http

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/dxngee/antifraud-processing/internal/config"
	"github.com/dxngee/antifraud-processing/internal/domain"
	"github.com/dxngee/antifraud-processing/internal/service"
	"github.com/dxngee/antifraud-processing/internal/utils"
)

type Handler struct {
	svc *service.FingerprintService
	cfg *config.Config
}

func NewHandler(svc *service.FingerprintService, cfg *config.Config) *Handler {
	return &Handler{svc: svc, cfg: cfg}
}

func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(corsMiddleware)

	r.Get("/health", h.health)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/fingerprints/resolve", h.resolveFingerprint)
	})

	return r
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) resolveFingerprint(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body", err.Error())
		return
	}

	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(body)))
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to decode base64 body", err.Error())
		return
	}

	input, err := utils.Unpack[domain.FingerprintInput](raw, h.cfg.PrivateKey)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to unpack body", err.Error())
		return
	}

	if input.AccountID == "" {
		writeError(w, http.StatusBadRequest, "account_id is required", "")
		return
	}

	result, err := h.svc.ResolveOrCreateProfile(r.Context(), *input)
	if err != nil {
		slog.Error("resolve failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error", "")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg, details string) {
	resp := domain.ErrorResponse{Error: msg, Details: details}
	writeJSON(w, status, resp)
}
