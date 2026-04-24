package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gofer/internal/delivery/http/middleware"
	"github.com/gofer/internal/domain"
	"github.com/gofer/internal/usecase/direct"
	"github.com/gofer/pkg/httputil"
)

type DirectHandler struct {
	directUC *direct.DirectUseCase
}

func NewDirectHandler(directUC *direct.DirectUseCase) *DirectHandler {
	return &DirectHandler{directUC: directUC}
}

func (h *DirectHandler) Start(w http.ResponseWriter, r *http.Request) {
	userCtx := r.Context().Value(middleware.UserIDKey).(*middleware.UserContext)
	targetUserID := r.PathValue("user_id")

	dm, err := h.directUC.StartDM(r.Context(), userCtx.UserID, targetUserID)
	if err != nil {
		if errors.Is(err, domain.ErrDirectChatAlreadyExists) {
			httputil.WriteError(w, http.StatusConflict, "direct chat already exists")
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
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(dm)
}

func (h *DirectHandler) History(w http.ResponseWriter, r *http.Request) {
	directID := r.PathValue("id")

	limitStr := r.URL.Query().Get("limit")
	beforeStr := r.URL.Query().Get("before")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 50
	}

	before, err := time.Parse(time.RFC3339, beforeStr)
	if err != nil {
		before = time.Now() // если не передали — берём текущее время
	}

	messages, err := h.directUC.GetDMHistory(r.Context(), directID, limit, before)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

func (h *DirectHandler) List(w http.ResponseWriter, r *http.Request) {
	userCtx := r.Context().Value(middleware.UserIDKey).(*middleware.UserContext)

	dms, err := h.directUC.ListDMs(r.Context(), userCtx.UserID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dms)
}

func (h *DirectHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userCtx := r.Context().Value(middleware.UserIDKey).(*middleware.UserContext)
	chatID := r.PathValue("id")

	err := h.directUC.DeleteDM(r.Context(), chatID, userCtx.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrDirectChatNotFound) {
			httputil.WriteError(w, http.StatusNotFound, "direct chat not found")
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			httputil.WriteError(w, http.StatusForbidden, "not a participant of this chat")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
