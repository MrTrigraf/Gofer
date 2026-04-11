package jwt

import (
	"fmt"
	"time"

	"github.com/gofer/pkg/config"
	"github.com/golang-jwt/jwt/v5"
)

type AccessClaims struct {
	UserID    string `json:"uid"`
	Username  string `json:"username"`
	TokenType string `json:"typ"`
	jwt.RegisteredClaims
}

type RefreshClaims struct {
	UserID    string `json:"uid"`
	TokenType string `json:"typ"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type TokenService struct {
	cfg *config.JWTConfig
}

func NewTokenService(cfg *config.JWTConfig) *TokenService {
	return &TokenService{cfg: cfg}
}

func (a AccessClaims) Validate() error {
	if a.TokenType != "access" {
		return fmt.Errorf("invalid token type: expected access, got %q", a.TokenType)
	}

	return nil
}

func (r RefreshClaims) Validate() error {
	if r.TokenType != "refresh" {
		return fmt.Errorf("invalid token type: expected refresh, got %q", r.TokenType)
	}
	return nil
}

func (s *TokenService) generateAccessToken(userID, username string) (string, error) {
	accessTTL, err := time.ParseDuration(s.cfg.AccessTTL)
	if err != nil {
		return "", fmt.Errorf("parse access ttl: %w", err)
	}

	now := time.Now()

	accessClaims := AccessClaims{
		UserID:    userID,
		Username:  username,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(s.cfg.Secret))
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}

	return token, nil
}

func (s *TokenService) generateRefreshToken(userID string) (string, error) {
	refreshTTL, err := time.ParseDuration(s.cfg.RefreshTTL)
	if err != nil {
		return "", fmt.Errorf("parse refresh ttl: %w", err)
	}

	now := time.Now()

	refreshClaims := RefreshClaims{
		UserID:    userID,
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(refreshTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString([]byte(s.cfg.Refresh))
	if err != nil {
		return "", fmt.Errorf("sign refresh token: %w", err)
	}

	return token, nil
}

func (s *TokenService) GenerateTokens(userID, username string) (TokenPair, error) {
	accessToken, err := s.generateAccessToken(userID, username)
	if err != nil {
		return TokenPair{}, fmt.Errorf("generate tokens: %w", err)
	}

	refreshToken, err := s.generateRefreshToken(userID)
	if err != nil {
		return TokenPair{}, fmt.Errorf("generate tokens: %w", err)
	}

	return TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *TokenService) ParseAccessToken(tokenString string) (*AccessClaims, error) {
	claims := &AccessClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.cfg.Secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse access token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is invalid")
	}

	return claims, nil
}

func (s *TokenService) ParseRefreshToken(tokenString string) (*RefreshClaims, error) {
	claims := &RefreshClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.cfg.Refresh), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse refresh token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is invalid")
	}

	return claims, nil
}
