package handlers

import (
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/japanesestudent/auth-service/internal/models"
	"github.com/japanesestudent/libs/handlers"
	"go.uber.org/zap"
)

// AuthService is the interface that wraps methods for authentication business logic.
type AuthService interface {
	// Method Register performs a user credentials validation and creation and returns access and refresh tokens.
	//
	// "req" parameter contains email, username and password.
	// "avatarFile" parameter is an optional file reader for the avatar image.
	// "avatarFilename" parameter is the name of the avatar image file.
	//
	// If user passed invalid credentials, or such user already exists, or some other error occurs, the error will be returned together with empty strings for access and refresh tokens.
	Register(ctx context.Context, req *models.RegisterRequest, avatarFile multipart.File, avatarFilename string) (string, string, error)
	// Method Login performs a user credentials validation and returns a user.
	//
	// "req" parameter contains login and password.
	//
	// If user passed invalid credentials, or such user does not exist, or some other error occurs, the error will be returned together with empty strings for access and refresh tokens.
	Login(ctx context.Context, req *models.LoginRequest) (string, string, error)
	// Method Refresh performs a refresh token validation and returns a new access token and refresh token.
	//
	// "refreshToken" parameter is used to identify the user.
	//
	// If refresh token is invalid or expired, or some other error occurs, the error will be returned together with empty strings for new access and refresh tokens.
	Refresh(ctx context.Context, refreshToken string) (string, string, error)
}

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	handlers.BaseHandler
	authService AuthService
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(
	authService AuthService,
	logger *zap.Logger,
) *AuthHandler {
	return &AuthHandler{
		BaseHandler: handlers.BaseHandler{Logger: logger},
		authService: authService,
	}
}

// RegisterRoutes registers all auth handler routes
// Note: This assumes the router is already scoped to /api/v6
func (h *AuthHandler) RegisterRoutes(r chi.Router) {
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)
		r.Post("/refresh", h.Refresh)
	})
}

// Register handles POST /auth/register
// @Summary Register a new user
// @Description Register a new user with email, username, password, and optional avatar. Returns access and refresh tokens as HTTP-only cookies.
// @Tags auth
// @Accept multipart/form-data
// @Produce json
// @Param email formData string true "User email"
// @Param username formData string true "Username"
// @Param password formData string true "User password"
// @Param avatar formData file false "User avatar image (optional)"
// @Success 201 {object} map[string]string "User registered successfully"
// @Failure 400 {object} map[string]string "Invalid request body or user already exists"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (limit to 20MB to match request size limit)
	err := r.ParseMultipartForm(20 << 20) // 20MB
	if err != nil {
		h.Logger.Error("failed to parse multipart form", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "failed to parse request")
		return
	}

	// Extract form values
	email := r.FormValue("email")
	username := r.FormValue("username")
	password := r.FormValue("password")

	if email == "" || username == "" || password == "" {
		h.RespondError(w, http.StatusBadRequest, "email, username, and password are required")
		return
	}

	req := &models.RegisterRequest{
		Email:    email,
		Username: username,
		Password: password,
	}

	// Extract avatar file (optional)
	var avatarFile multipart.File
	var avatarFilename string
	file, fileHeader, err := r.FormFile("avatar")
	if err == nil && file != nil && fileHeader != nil {
		// Validate file is actually provided (not just empty field)
		if fileHeader.Size > 0 {
			avatarFile = file
			avatarFilename = fileHeader.Filename
			defer file.Close()
		}
	} else if err != http.ErrMissingFile {
		// If error is not "missing file", it's a real error
		h.Logger.Error("failed to get avatar file from form", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "failed to process avatar file")
		return
	}

	// Register user
	accessToken, refreshToken, err := h.authService.Register(r.Context(), req, avatarFile, avatarFilename)
	if err != nil {
		h.Logger.Error("failed to register user", zap.Error(err))
		errStatus := http.StatusInternalServerError
		// Return appropriate status code based on error
		if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "generate") || strings.Contains(err.Error(), "check") {
			errStatus = http.StatusBadRequest
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	// Set cookies
	h.setTokenCookies(w, accessToken, refreshToken)

	// Return success response
	h.RespondJSON(w, http.StatusCreated, map[string]string{"message": "user registered successfully"})
}

// Login handles POST /auth/login
// @Summary Login user
// @Description Authenticate user with login (email or username) and password. Returns access and refresh tokens as HTTP-only cookies.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.LoginRequest true "Login request"
// @Success 200 {object} map[string]string "Login successful"
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 401 {object} map[string]string "Invalid credentials"
// @Router /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Authenticate user
	accessToken, refreshToken, err := h.authService.Login(r.Context(), &req)
	if err != nil {
		h.Logger.Error("failed to login user", zap.Error(err))
		h.RespondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	// Set cookies
	h.setTokenCookies(w, accessToken, refreshToken)

	// Return success response
	h.RespondJSON(w, http.StatusOK, map[string]string{"message": "login successful"})
}

// RefreshRequest represents a token refresh request
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Refresh handles POST /auth/refresh
// @Summary Refresh access token
// @Description Refresh access and refresh tokens using a valid refresh token. Token can be provided in request body or as a cookie.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RefreshRequest false "Refresh token request (optional if using cookie)"
// @Success 200 {object} map[string]string "Tokens refreshed successfully"
// @Failure 400 {object} map[string]string "Refresh token required"
// @Failure 500 {object} map[string]string "Failed to refresh tokens"
// @Router /auth/refresh [post]
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	// Get refresh token from request body or cookie
	var refreshToken string
	var req RefreshRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err == nil && req.RefreshToken != "" {
		refreshToken = req.RefreshToken
	} else {
		cookie, err := r.Cookie("refresh_token")
		if err != nil {
			h.Logger.Error("failed to get refresh token from cookie", zap.Error(err))
			h.RespondError(w, http.StatusBadRequest, "refresh token required")
			return
		}
		refreshToken = cookie.Value
	}

	// Refresh tokens
	accessToken, newRefreshToken, err := h.authService.Refresh(r.Context(), refreshToken)
	if err != nil {
		h.Logger.Error("failed to refresh tokens", zap.Error(err))
		h.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Set cookies
	h.setTokenCookies(w, accessToken, newRefreshToken)

	// Return success response
	h.RespondJSON(w, http.StatusOK, map[string]string{"message": "tokens refreshed successfully"})
}

// setTokenCookies sets access and refresh tokens as HTTP-only cookies
func (h *AuthHandler) setTokenCookies(w http.ResponseWriter, accessToken, refreshToken string) {
	// Access token cookie (1 hour)
	accessCookie := &http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Path:     "/",
		MaxAge:   3600, // 1 hour
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, accessCookie)

	// Refresh token cookie (7 days)
	refreshCookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		MaxAge:   604800, // 7 days
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, refreshCookie)
}
