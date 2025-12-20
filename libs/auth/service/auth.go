package service

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenGenerator handles JWT token generation and validation
type TokenGenerator struct {
	secret             string
	accessTokenExpiry  time.Duration
	refreshTokenExpiry time.Duration
}

// NewTokenGenerator creates a new token generator
func NewTokenGenerator(secret string, accessExpiry, refreshExpiry time.Duration) *TokenGenerator {
	return &TokenGenerator{
		secret:             secret,
		accessTokenExpiry:  accessExpiry,
		refreshTokenExpiry: refreshExpiry,
	}
}

// GenerateTokens generates both access and refresh tokens for a user
// Access token contains user_id and role in payload, refresh token does not
func (tg *TokenGenerator) GenerateTokens(userID int, role int) (string, string, error) {
	// Generate access token with userID and role
	accessToken, err := tg.generateAccessToken(userID, role)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token without userID
	refreshToken, err := tg.generateRefreshToken()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

// generateAccessToken creates an access token with userID and role in payload
func (tg *TokenGenerator) generateAccessToken(userID int, role int) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"exp":     time.Now().Add(tg.accessTokenExpiry).Unix(),
		"iat":     time.Now().Unix(),
		"type":    "access",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(tg.secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign access token: %w", err)
	}

	return tokenString, nil
}

// generateRefreshToken creates a refresh token without userID
func (tg *TokenGenerator) generateRefreshToken() (string, error) {
	claims := jwt.MapClaims{
		"exp":  time.Now().Add(tg.refreshTokenExpiry).Unix(),
		"iat":  time.Now().Unix(),
		"type": "refresh",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(tg.secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return tokenString, nil
}

// ValidateAccessToken validates an access token and returns the userID and role
func (tg *TokenGenerator) ValidateAccessToken(tokenString string) (int, int, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(tg.secret), nil
	})

	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return 0, 0, fmt.Errorf("token is invalid")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, 0, fmt.Errorf("invalid token claims")
	}

	// Check token type
	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != "access" {
		return 0, 0, fmt.Errorf("token is not an access token")
	}

	// Extract userID (JWT claims decode numbers as float64)
	userIDInt, ok := claims["user_id"].(float64)
	if !ok {
		return 0, 0, fmt.Errorf("user_id not found in token")
	}

	// Extract role (JWT claims decode numbers as float64)
	roleInt, ok := claims["role"].(float64)
	if !ok {
		return 0, 0, fmt.Errorf("role not found in token")
	}

	return int(userIDInt), int(roleInt), nil
}

// ValidateRefreshToken validates a refresh token
func (tg *TokenGenerator) ValidateRefreshToken(tokenString string) error {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(tg.secret), nil
	})

	if err != nil {
		return fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return fmt.Errorf("token is invalid")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return fmt.Errorf("invalid token claims")
	}

	// Check token type
	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != "refresh" {
		return fmt.Errorf("token is not a refresh token")
	}

	return nil
}
