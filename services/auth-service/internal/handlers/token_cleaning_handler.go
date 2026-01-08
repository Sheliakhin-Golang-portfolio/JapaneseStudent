package handlers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/japanesestudent/auth-service/internal/services"
	"github.com/japanesestudent/libs/handlers"
	"go.uber.org/zap"
)

// TokenCleaningHandler handles token cleaning requests
type TokenCleaningHandler struct {
	handlers.BaseHandler
	userTokenRepo      services.UserTokenRepository
	refreshTokenExpiry time.Duration
}

// NewTokenCleaningHandler creates a new token cleaning handler
func NewTokenCleaningHandler(
	userTokenRepo services.UserTokenRepository,
	logger *zap.Logger,
	refreshTokenExpiry time.Duration,
) *TokenCleaningHandler {
	return &TokenCleaningHandler{
		BaseHandler:        handlers.BaseHandler{Logger: logger},
		userTokenRepo:      userTokenRepo,
		refreshTokenExpiry: refreshTokenExpiry,
	}
}

// RegisterRoutes registers token cleaning handler routes
func (h *TokenCleaningHandler) RegisterRoutes(r chi.Router) {
	r.Get("/tokens/clean", h.CleanTokens)
}

// CleanTokens handles GET /tokens/clean
// @Summary Clean expired tokens
// @Description Removes all user tokens with created_at older than refresh token expiry time
// @Tags tokens
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]string "Token cleaning completed successfully"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /tokens/clean [get]
func (h *TokenCleaningHandler) CleanTokens(w http.ResponseWriter, r *http.Request) {
	// Calculate expiry time: current time minus refresh token expiry
	expiryTime := time.Now().Add(-h.refreshTokenExpiry)

	// Delete expired tokens
	deletedCount, err := h.userTokenRepo.DeleteExpiredTokens(r.Context(), expiryTime)
	if err != nil {
		h.Logger.Error("failed to delete expired tokens", zap.Error(err))
		h.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Log success (0 deleted rows is not an error)
	h.Logger.Info("token cleaning completed successfully", zap.Int("deletedCount", deletedCount))
	h.RespondJSON(w, http.StatusOK, map[string]string{"message": "token cleaning completed successfully"})
}
