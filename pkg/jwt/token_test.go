package jwt

import (
	"testing"
	"time"

	"github.com/gofer/pkg/config"
	jwtlib "github.com/golang-jwt/jwt/v5"
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

// testCfg — общий конфиг для тестов парсинга.
func testCfg() *config.JWTConfig {
	return &config.JWTConfig{
		Secret:     "test-secret",
		Refresh:    "test-refresh",
		AccessTTL:  "15m",
		RefreshTTL: "168h",
	}
}

func makeAccessToken(t *testing.T, secret string, exp time.Time) string {
	t.Helper()
	claims := AccessClaims{
		UserID:    "123",
		Username:  "lol",
		TokenType: "access",
		RegisteredClaims: jwtlib.RegisteredClaims{
			ExpiresAt: jwtlib.NewNumericDate(exp),
			IssuedAt:  jwtlib.NewNumericDate(time.Now()),
		},
	}
	s, err := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims).SignedString([]byte(secret))
	require.NoError(t, err)
	return s
}

func TestParseAccessToken_WrongSecret(t *testing.T) {
	svc := NewTokenService(testCfg())

	// Валиден по времени, но подписан не тем секретом.
	tokenStr := makeAccessToken(t, "attacker-secret", time.Now().Add(15*time.Minute))

	_, err := svc.ParseAccessToken(tokenStr)
	require.Error(t, err)
	assert.ErrorIs(t, err, jwtlib.ErrTokenSignatureInvalid)
}

func TestParseAccessToken_Expired(t *testing.T) {
	svc := NewTokenService(testCfg())

	// Правильный секрет, срок в прошлом.
	tokenStr := makeAccessToken(t, "test-secret", time.Now().Add(-1*time.Minute))

	_, err := svc.ParseAccessToken(tokenStr)
	require.Error(t, err)
	assert.ErrorIs(t, err, jwtlib.ErrTokenExpired)
}

func TestParseAccessToken_ExpiredAndWrongSecret(t *testing.T) {
	svc := NewTokenService(testCfg())

	// И протух, и подписан чужим секретом — худший случай.
	tokenStr := makeAccessToken(t, "attacker-secret", time.Now().Add(-1*time.Minute))

	_, err := svc.ParseAccessToken(tokenStr)
	require.Error(t, err)
	// Ключевое: подпись невалидна и это НЕ должно потеряться за "expired".
	assert.ErrorIs(t, err, jwtlib.ErrTokenSignatureInvalid)
}
