package handlers

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

type BaseHandler struct {
	logger *zap.Logger
}

// respondJSON sends a JSON response
func (h *BaseHandler) respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode JSON response", zap.Error(err))
	}
}

// respondError sends an error JSON response
func (h *BaseHandler) respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
