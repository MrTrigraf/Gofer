package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gofer/internal/usecase"
	"github.com/gofer/pkg/httputil"
)

type contextKey string

const UserIDKey contextKey = "userID"

type UserContext struct {
	UserID   string
	Username string
}

type JWTMiddleware struct {
	tokenService usecase.TokenService
}

func NewJWTMiddleware(tokenService usecase.TokenService) *JWTMiddleware {
	return &JWTMiddleware{tokenService: tokenService}
}

func (m *JWTMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			httputil.WriteError(w, http.StatusUnauthorized, "missing authorization token")
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			httputil.WriteError(w, http.StatusUnauthorized, "invalid token format")
			return
		}

		tokenString := parts[1]

		claims, err := m.tokenService.ParseAccessToken(tokenString)
		if err != nil {
			httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		userCtx := &UserContext{
			UserID:   claims.UserID,
			Username: claims.Username,
		}

		ctx := context.WithValue(r.Context(), UserIDKey, userCtx)
		next(w, r.WithContext(ctx))
	}
}
