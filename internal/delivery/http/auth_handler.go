package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gofer/internal/domain"
	"github.com/gofer/internal/usecase/auth"
	"github.com/gofer/pkg/httputil"
)

type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserResponse struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
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
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.authUC.Register(r.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrUserAlreadyExists) {
			httputil.WriteError(w, http.StatusConflict, "user already exists")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		CreatedAt: user.CreatedAt,
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tokens, err := h.authUC.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			httputil.WriteError(w, http.StatusNotFound, "user not found")
			return
		}
		if errors.Is(err, domain.ErrInvalidCredentials) {
			httputil.WriteError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tokens)
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
