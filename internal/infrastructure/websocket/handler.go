package websocket

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gofer/internal/usecase"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:   4096,
	WriteBufferSize:  4096,
	HandshakeTimeout: 10 * time.Second,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	EnableCompression: false,
}

type Handler struct {
	hub          *Hub
	tokenService usecase.TokenService
}

func NewHandler(hub *Hub, tokenService usecase.TokenService) *Handler {
	return &Handler{
		hub:          hub,
		tokenService: tokenService,
	}
}

func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	claims, err := h.tokenService.ParseAccessToken(tokenString)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("upgrade error", "user", claims.Username, "err", err)
		return
	}

	client := NewClient(h.hub, conn, claims.UserID, claims.Username)

	h.hub.register <- client

	go client.writePump()
	go client.readPump()
}
