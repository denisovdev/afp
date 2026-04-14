package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/dxngee/antifraud-processing/internal/domain"
	"github.com/dxngee/antifraud-processing/internal/service"
)

type Handler struct {
	svc *service.FingerprintService
}

func NewHandler(svc *service.FingerprintService) *Handler {
	return &Handler{svc: svc}
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
	var input domain.FingerprintInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body", err.Error())
		return
	}

	if input.AccountID <= 0 {
		writeError(w, http.StatusBadRequest, "account_id must be > 0", "")
		return
	}

	result, err := h.svc.ResolveOrCreateProfile(r.Context(), input)
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
