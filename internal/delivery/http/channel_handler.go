package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gofer/internal/delivery/http/middleware"
	"github.com/gofer/internal/domain"
	"github.com/gofer/internal/usecase/channel"
	"github.com/gofer/pkg/httputil"
)

type CreateChannelRequest struct {
	Name string `json:"name"`
}

type ChannelHandler struct {
	channelUC *channel.ChannelUseCase
}

func NewChannelHandler(channelUC *channel.ChannelUseCase) *ChannelHandler {
	return &ChannelHandler{channelUC: channelUC}
}

func (h *ChannelHandler) List(w http.ResponseWriter, r *http.Request) {
	userCtx := r.Context().Value(middleware.UserIDKey).(*middleware.UserContext)

	channels, err := h.channelUC.ListChannels(r.Context(), userCtx.UserID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(channels)
}

func (h *ChannelHandler) Create(w http.ResponseWriter, r *http.Request) {
	userCtx := r.Context().Value(middleware.UserIDKey).(*middleware.UserContext)

	var req CreateChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ch, err := h.channelUC.CreateChannel(r.Context(), req.Name, userCtx.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrChannelAlreadyExists) {
			httputil.WriteError(w, http.StatusConflict, "channel already exists")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ch)
}

func (h *ChannelHandler) Join(w http.ResponseWriter, r *http.Request) {
	userCtx := r.Context().Value(middleware.UserIDKey).(*middleware.UserContext)
	channelID := r.PathValue("id")

	err := h.channelUC.JoinChannel(r.Context(), channelID, userCtx.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrGroupNotFound) {
			httputil.WriteError(w, http.StatusNotFound, "channel not found")
			return
		}
		if errors.Is(err, domain.ErrUserNotFound) {
			httputil.WriteError(w, http.StatusNotFound, "user not found")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *ChannelHandler) History(w http.ResponseWriter, r *http.Request) {
	channelID := r.PathValue("id")

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

	messages, err := h.channelUC.GetChannelHistory(r.Context(), channelID, limit, before)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}
