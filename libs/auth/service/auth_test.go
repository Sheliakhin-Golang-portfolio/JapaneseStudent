package service

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTokenGenerator(t *testing.T) {
	tests := []struct {
		name           string
		secret         string
		accessExpiry   time.Duration
		refreshExpiry  time.Duration
		expectedSecret string
	}{
		{
			name:           "standard initialization",
			secret:         "test-secret-key",
			accessExpiry:   1 * time.Hour,
			refreshExpiry:  7 * 24 * time.Hour,
			expectedSecret: "test-secret-key",
		},
		{
			name:           "short expiry times",
			secret:         "short-secret",
			accessExpiry:   1 * time.Minute,
			refreshExpiry:  10 * time.Minute,
			expectedSecret: "short-secret",
		},
		{
			name:           "long expiry times",
			secret:         "long-secret",
			accessExpiry:   24 * time.Hour,
			refreshExpiry:  30 * 24 * time.Hour,
			expectedSecret: "long-secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tg := NewTokenGenerator(tt.secret, tt.accessExpiry, tt.refreshExpiry)

			assert.NotNil(t, tg)
			assert.Equal(t, tt.expectedSecret, tg.secret)
			assert.Equal(t, tt.accessExpiry, tg.accessTokenExpiry)
			assert.Equal(t, tt.refreshExpiry, tg.refreshTokenExpiry)
		})
	}
}

func TestTokenGenerator_GenerateTokens(t *testing.T) {
	secret := "b8a3c2267dc85f855dea9b46b452bf20"
	accessExpiry := 1 * time.Hour
	refreshExpiry := 7 * 24 * time.Hour

	tg := NewTokenGenerator(secret, accessExpiry, refreshExpiry)

	t.Run("success with standard userID", func(t *testing.T) {
		userID := 123
		accessToken, refreshToken, err := tg.GenerateTokens(userID, 1)
		require.NoError(t, err)
		assert.NotEmpty(t, accessToken)
		assert.NotEmpty(t, refreshToken)
		assert.NotEqual(t, accessToken, refreshToken)
	})

	t.Run("userID zero", func(t *testing.T) {
		accessToken, refreshToken, err := tg.GenerateTokens(0, 1)
		require.NoError(t, err)
		assert.NotEmpty(t, accessToken)
		assert.NotEmpty(t, refreshToken)

		// Verify userID 0 is in the token
		userID, _, err := tg.ValidateAccessToken(accessToken)
		require.NoError(t, err)
		assert.Equal(t, 0, userID)
	})

	t.Run("negative userID", func(t *testing.T) {
		accessToken, refreshToken, err := tg.GenerateTokens(-1, 1)
		require.NoError(t, err)
		assert.NotEmpty(t, accessToken)
		assert.NotEmpty(t, refreshToken)
	})

	t.Run("max int userID", func(t *testing.T) {
		accessToken, refreshToken, err := tg.GenerateTokens(math.MaxInt32, 1)
		require.NoError(t, err)
		assert.NotEmpty(t, accessToken)
		assert.NotEmpty(t, refreshToken)

		userID, _, err := tg.ValidateAccessToken(accessToken)
		require.NoError(t, err)
		assert.Equal(t, math.MaxInt32, userID)
	})

	t.Run("token uniqueness", func(t *testing.T) {
		userID := 456
		token1Access, token1Refresh, err := tg.GenerateTokens(userID, 1)
		require.NoError(t, err)

		// Wait to ensure different iat timestamp
		time.Sleep(1 * time.Second)

		// Generate again with same userID
		token2Access, token2Refresh, err := tg.GenerateTokens(userID, 1)
		require.NoError(t, err)

		// Tokens should be different even for same userID (due to iat timestamp)
		assert.NotEqual(t, token1Access, token2Access)
		assert.NotEqual(t, token1Refresh, token2Refresh)
	})

	t.Run("token format validation", func(t *testing.T) {
		accessToken, refreshToken, err := tg.GenerateTokens(789, 1)
		require.NoError(t, err)

		// JWT tokens should have 3 parts separated by dots
		accessParts := strings.Split(accessToken, ".")
		assert.Len(t, accessParts, 3)

		refreshParts := strings.Split(refreshToken, ".")
		assert.Len(t, refreshParts, 3)
	})
}

func TestTokenGenerator_ValidateAccessToken(t *testing.T) {
	secret := "b8a3c2267dc85f855dea9b46b452bf20"
	accessExpiry := 1 * time.Hour
	refreshExpiry := 7 * 24 * time.Hour

	tg := NewTokenGenerator(secret, accessExpiry, refreshExpiry)

	t.Run("valid token", func(t *testing.T) {
		userID := 456
		accessToken, _, err := tg.GenerateTokens(userID, 1)
		require.NoError(t, err)

		validatedUserID, _, err := tg.ValidateAccessToken(accessToken)
		require.NoError(t, err)
		assert.Equal(t, userID, validatedUserID)
	})

	t.Run("empty string token", func(t *testing.T) {
		_, _, err := tg.ValidateAccessToken("")
		assert.Error(t, err)
	})

	t.Run("invalid token format", func(t *testing.T) {
		_, _, err := tg.ValidateAccessToken("invalid-token")
		assert.Error(t, err)
	})

	t.Run("malformed JWT - missing parts", func(t *testing.T) {
		_, _, err := tg.ValidateAccessToken("header.payload")
		assert.Error(t, err)
	})

	t.Run("malformed JWT - invalid base64", func(t *testing.T) {
		_, _, err := tg.ValidateAccessToken("not-base64.not-base64.not-base64")
		assert.Error(t, err)
	})

	t.Run("wrong signature method - non-HMAC", func(t *testing.T) {
		// Create a token with None signing method (not HMAC)
		// This should be rejected by the validator
		claims := jwt.MapClaims{
			"user_id": 123,
			"exp":     time.Now().Add(1 * time.Hour).Unix(),
			"iat":     time.Now().Unix(),
			"type":    "access",
		}
		token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
		tokenString, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
		require.NoError(t, err)

		_, _, err = tg.ValidateAccessToken(tokenString)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected signing method")
	})

	t.Run("token without user_id claim", func(t *testing.T) {
		claims := jwt.MapClaims{
			"exp":  time.Now().Add(1 * time.Hour).Unix(),
			"iat":  time.Now().Unix(),
			"type": "access",
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(secret))
		require.NoError(t, err)

		_, _, err = tg.ValidateAccessToken(tokenString)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user_id not found")
	})

	t.Run("token without type claim", func(t *testing.T) {
		claims := jwt.MapClaims{
			"user_id": 123,
			"exp":     time.Now().Add(1 * time.Hour).Unix(),
			"iat":     time.Now().Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(secret))
		require.NoError(t, err)

		_, _, err = tg.ValidateAccessToken(tokenString)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not an access token")
	})

	t.Run("token with wrong type - refresh instead of access", func(t *testing.T) {
		claims := jwt.MapClaims{
			"user_id": 123,
			"exp":     time.Now().Add(1 * time.Hour).Unix(),
			"iat":     time.Now().Unix(),
			"type":    "refresh",
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(secret))
		require.NoError(t, err)

		_, _, err = tg.ValidateAccessToken(tokenString)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not an access token")
	})

	t.Run("token with string user_id", func(t *testing.T) {
		claims := jwt.MapClaims{
			"user_id": "not-a-number",
			"exp":     time.Now().Add(1 * time.Hour).Unix(),
			"iat":     time.Now().Unix(),
			"type":    "access",
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(secret))
		require.NoError(t, err)

		_, _, err = tg.ValidateAccessToken(tokenString)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user_id not found")
	})

	t.Run("expired token", func(t *testing.T) {
		claims := jwt.MapClaims{
			"user_id": 123,
			"exp":     time.Now().Add(-1 * time.Hour).Unix(),
			"iat":     time.Now().Add(-2 * time.Hour).Unix(),
			"type":    "access",
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(secret))
		require.NoError(t, err)

		_, _, err = tg.ValidateAccessToken(tokenString)
		assert.Error(t, err)
	})

	t.Run("wrong secret", func(t *testing.T) {
		userID := 789
		accessToken, _, err := tg.GenerateTokens(userID, 1)
		require.NoError(t, err)

		wrongTG := NewTokenGenerator("wrong-secret", accessExpiry, refreshExpiry)
		_, _, err = wrongTG.ValidateAccessToken(accessToken)
		assert.Error(t, err)
	})
}

func TestTokenGenerator_ValidateRefreshToken(t *testing.T) {
	secret := "b8a3c2267dc85f855dea9b46b452bf20"
	accessExpiry := 1 * time.Hour
	refreshExpiry := 7 * 24 * time.Hour

	tg := NewTokenGenerator(secret, accessExpiry, refreshExpiry)

	t.Run("valid refresh token", func(t *testing.T) {
		_, refreshToken, err := tg.GenerateTokens(789, 1)
		require.NoError(t, err)

		err = tg.ValidateRefreshToken(refreshToken)
		assert.NoError(t, err)
	})

	t.Run("empty string token", func(t *testing.T) {
		err := tg.ValidateRefreshToken("")
		assert.Error(t, err)
	})

	t.Run("invalid token format", func(t *testing.T) {
		err := tg.ValidateRefreshToken("invalid-token")
		assert.Error(t, err)
	})

	t.Run("malformed JWT - missing parts", func(t *testing.T) {
		err := tg.ValidateRefreshToken("header.payload")
		assert.Error(t, err)
	})

	t.Run("malformed JWT - invalid base64", func(t *testing.T) {
		err := tg.ValidateRefreshToken("not-base64.not-base64.not-base64")
		assert.Error(t, err)
	})

	t.Run("wrong signature method - non-HMAC", func(t *testing.T) {
		claims := jwt.MapClaims{
			"exp":  time.Now().Add(1 * time.Hour).Unix(),
			"iat":  time.Now().Unix(),
			"type": "refresh",
		}
		token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
		tokenString, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
		require.NoError(t, err)

		err = tg.ValidateRefreshToken(tokenString)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected signing method")
	})

	t.Run("token without type claim", func(t *testing.T) {
		claims := jwt.MapClaims{
			"exp": time.Now().Add(1 * time.Hour).Unix(),
			"iat": time.Now().Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(secret))
		require.NoError(t, err)

		err = tg.ValidateRefreshToken(tokenString)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not a refresh token")
	})

	t.Run("access token used as refresh token", func(t *testing.T) {
		accessToken, _, err := tg.GenerateTokens(789, 1)
		require.NoError(t, err)

		err = tg.ValidateRefreshToken(accessToken)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not a refresh token")
	})

	t.Run("expired refresh token", func(t *testing.T) {
		claims := jwt.MapClaims{
			"exp":  time.Now().Add(-1 * time.Hour).Unix(),
			"iat":  time.Now().Add(-2 * time.Hour).Unix(),
			"type": "refresh",
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(secret))
		require.NoError(t, err)

		err = tg.ValidateRefreshToken(tokenString)
		assert.Error(t, err)
	})

	t.Run("wrong secret", func(t *testing.T) {
		_, refreshToken, err := tg.GenerateTokens(999, 1)
		require.NoError(t, err)

		wrongTG := NewTokenGenerator("wrong-secret", accessExpiry, refreshExpiry)
		err = wrongTG.ValidateRefreshToken(refreshToken)
		assert.Error(t, err)
	})
}

func TestTokenGenerator_TokenExpiry(t *testing.T) {
	secret := "b8a3c2267dc85f855dea9b46b452bf20"
	accessExpiry := 1 * time.Second
	refreshExpiry := 7 * 24 * time.Hour

	tg := NewTokenGenerator(secret, accessExpiry, refreshExpiry)

	accessToken, _, err := tg.GenerateTokens(123, 1)
	require.NoError(t, err)

	// Token should be valid immediately
	_, _, err = tg.ValidateAccessToken(accessToken)
	require.NoError(t, err)

	// Wait for token to expire (wait longer than the expiry time)
	time.Sleep(1200 * time.Millisecond)

	// Token should be invalid after expiry
	_, _, err = tg.ValidateAccessToken(accessToken)
	assert.Error(t, err)
}

func TestTokenGenerator_TokenClaims(t *testing.T) {
	secret := "b8a3c2267dc85f855dea9b46b452bf20"
	accessExpiry := 1 * time.Hour
	refreshExpiry := 7 * 24 * time.Hour

	tg := NewTokenGenerator(secret, accessExpiry, refreshExpiry)

	t.Run("access token claims", func(t *testing.T) {
		userID := 123
		beforeGeneration := time.Now().Unix()
		accessToken, _, err := tg.GenerateTokens(userID, 1)
		require.NoError(t, err)
		afterGeneration := time.Now().Unix()

		// Parse token to check claims
		token, err := jwt.Parse(accessToken, func(token *jwt.Token) (any, error) {
			return []byte(secret), nil
		})
		require.NoError(t, err)
		require.True(t, token.Valid)

		claims, ok := token.Claims.(jwt.MapClaims)
		require.True(t, ok)

		// Verify user_id is present and correct
		userIDFloat, ok := claims["user_id"].(float64)
		require.True(t, ok)
		assert.Equal(t, userID, int(userIDFloat))

		// Verify type is "access"
		tokenType, ok := claims["type"].(string)
		require.True(t, ok)
		assert.Equal(t, "access", tokenType)

		// Verify iat is set correctly (within reasonable time window)
		iat, ok := claims["iat"].(float64)
		require.True(t, ok)
		assert.GreaterOrEqual(t, int64(iat), beforeGeneration)
		assert.LessOrEqual(t, int64(iat), afterGeneration)

		// Verify exp is set correctly
		exp, ok := claims["exp"].(float64)
		require.True(t, ok)
		expectedExp := time.Unix(int64(iat), 0).Add(accessExpiry).Unix()
		assert.Equal(t, expectedExp, int64(exp))
	})

	t.Run("refresh token claims", func(t *testing.T) {
		userID := 456
		beforeGeneration := time.Now().Unix()
		_, refreshToken, err := tg.GenerateTokens(userID, 1)
		require.NoError(t, err)
		afterGeneration := time.Now().Unix()

		// Parse token to check claims
		token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (any, error) {
			return []byte(secret), nil
		})
		require.NoError(t, err)
		require.True(t, token.Valid)

		claims, ok := token.Claims.(jwt.MapClaims)
		require.True(t, ok)

		// Verify user_id is NOT present in refresh token
		_, hasUserID := claims["user_id"]
		assert.False(t, hasUserID, "refresh token should not contain user_id")

		// Verify type is "refresh"
		tokenType, ok := claims["type"].(string)
		require.True(t, ok)
		assert.Equal(t, "refresh", tokenType)

		// Verify iat is set correctly
		iat, ok := claims["iat"].(float64)
		require.True(t, ok)
		assert.GreaterOrEqual(t, int64(iat), beforeGeneration)
		assert.LessOrEqual(t, int64(iat), afterGeneration)

		// Verify exp is set correctly
		exp, ok := claims["exp"].(float64)
		require.True(t, ok)
		expectedExp := time.Unix(int64(iat), 0).Add(refreshExpiry).Unix()
		assert.Equal(t, expectedExp, int64(exp))
	})

	t.Run("different tokens have different iat", func(t *testing.T) {
		_, refresh1, err := tg.GenerateTokens(789, 1)
		require.NoError(t, err)

		// Wait to ensure different iat timestamp (Unix timestamps are in seconds)
		time.Sleep(1 * time.Second)

		_, refresh2, err := tg.GenerateTokens(789, 1)
		require.NoError(t, err)

		// Tokens should be different due to different iat timestamps
		assert.NotEqual(t, refresh1, refresh2)
	})
}
