package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gofer/internal/domain"
	"github.com/gofer/internal/usecase/auth"
	"github.com/gofer/pkg/httputil"
	"github.com/gofer/pkg/jwt"
)

type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	User   domain.User   `json:"user"`
	Tokens jwt.TokenPair `json:"tokens"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type AuthHandler struct {
	authUC *auth.AuthUseCase
}

func NewAuthHandler(authUC *auth.AuthUseCase) *AuthHandler {
	return &AuthHandler{authUC: authUC}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	slog.Info("register called")
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.authUC.Register(r.Context(), req.Username, req.Password)
	if err != nil {
		// 409 — конфликт: юзер уже есть
		if errors.Is(err, domain.ErrUserAlreadyExists) {
			httputil.WriteError(w, http.StatusConflict, "user already exists")
			return
		}
		// 400 — некорректный username
		if errors.Is(err, domain.ErrUsernameIsLong) {
			httputil.WriteError(w, http.StatusBadRequest, "username must be 1..16 characters")
			return
		}
		// 400 — некорректный password
		if errors.Is(err, domain.ErrPasswordTooShort) {
			httputil.WriteError(w, http.StatusBadRequest, "password must be at least 6 characters")
			return
		}
		if errors.Is(err, domain.ErrPasswordTooLong) {
			httputil.WriteError(w, http.StatusBadRequest, "password must be at most 64 characters")
			return
		}
		// fallback — 500
		slog.Error("register: internal error", "err", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, tokens, err := h.authUC.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			httputil.WriteError(w, http.StatusNotFound, "user not found")
			return
		}
		if errors.Is(err, domain.ErrInvalidCredentials) {
			httputil.WriteError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		slog.Error("login: internal error", "err", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(LoginResponse{
		User:   user,
		Tokens: tokens,
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	token, err := h.authUC.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, domain.ErrTokenInvalid) {
			httputil.WriteError(w, http.StatusUnauthorized, "invalid token")
			return
		}
		if errors.Is(err, domain.ErrUserNotFound) {
			httputil.WriteError(w, http.StatusNotFound, "user not found")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(token)
}
