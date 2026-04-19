package jwt

import (
	"testing"

	"github.com/gofer/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateTokens(t *testing.T) {
	cfg := &config.JWTConfig{
		Secret:     "test-secret",
		Refresh:    "test-refresh",
		AccessTTL:  "15m",
		RefreshTTL: "168h",
	}
	svc := NewTokenService(cfg)

	tokens, err := svc.GenerateTokens("123", "lol")

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)
}

func TestParseAccessToken(t *testing.T) {
	cfg := &config.JWTConfig{
		Secret:     "test-secret",
		Refresh:    "test-refresh",
		AccessTTL:  "15m",
		RefreshTTL: "168h",
	}
	svc := NewTokenService(cfg)
	tokens, err := svc.GenerateTokens("123", "lol")
	require.NoError(t, err)

	claims, err := svc.ParseAccessToken(tokens.AccessToken)

	require.NoError(t, err)
	assert.Equal(t, "123", claims.UserID)
	assert.Equal(t, "lol", claims.Username)
}

func TestParseRefreshToken(t *testing.T) {
	cfg := &config.JWTConfig{
		Secret:     "test-secret",
		Refresh:    "test-refresh",
		AccessTTL:  "15m",
		RefreshTTL: "168h",
	}
	svc := NewTokenService(cfg)
	tokens, err := svc.GenerateTokens("123", "lol")
	require.NoError(t, err)

	claims, err := svc.ParseRefreshToken(tokens.RefreshToken)

	require.NoError(t, err)
	assert.Equal(t, "123", claims.UserID)
}
