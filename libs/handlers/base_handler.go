package handlers

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

// BaseHandler provides common handler functionality
type BaseHandler struct {
	Logger *zap.Logger
}

// RespondJSON sends a JSON response
func (h *BaseHandler) RespondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.Logger.Error("failed to encode JSON response", zap.Error(err))
	}
}

// RespondError sends an error JSON response
func (h *BaseHandler) RespondError(w http.ResponseWriter, status int, message string) {
	h.RespondJSON(w, status, map[string]string{"error": message})
}
