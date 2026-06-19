package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/gofer/internal/domain"
	"github.com/gofer/internal/usecase"
)

const dbTimeout = 5 * time.Second

const pushBufferSize = 256

type IncomingMessage struct {
	client *Client
	data   []byte
}

type incomingWSMessage struct {
	Type     string `json:"type"`
	TargetID string `json:"target_id"`
	Content  string `json:"content"`
}

type outgoingWSMessage struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	TargetType string    `json:"target_type"`
	TargetID   string    `json:"target_id"`
	Content    string    `json:"content"`
	SenderID   string    `json:"sender_id"`
	Username   string    `json:"username"`
	CreatedAt  time.Time `json:"created_at"`
}

type outgoingEvent struct {
	Type string `json:"type"`
}

type pushMsg struct {
	userID  string
	payload []byte
}

const (
	msgTypeChannel   = "channel_message"
	msgTypeDM        = "dm_message"
	msgTypeDMCreated = "dm_created"
)

type Hub struct {
	clients     map[string]*Client
	broadcast   chan IncomingMessage
	register    chan *Client
	unregister  chan *Client
	push        chan pushMsg
	channelRepo usecase.ChannelRepository
	messageRepo usecase.MessageRepository
	directRepo  usecase.DirectChatRepository
}

func NewHub(
	channelRepo usecase.ChannelRepository,
	messageRepo usecase.MessageRepository,
	directRepo usecase.DirectChatRepository,
) *Hub {
	return &Hub{
		clients:     make(map[string]*Client),
		broadcast:   make(chan IncomingMessage, 256),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		push:        make(chan pushMsg, pushBufferSize),
		channelRepo: channelRepo,
		messageRepo: messageRepo,
		directRepo:  directRepo,
	}
}

func (h *Hub) Run(ctx context.Context) {
	defer h.shutdown()

	for {
		select {
		case <-ctx.Done():
			return

		case client := <-h.register:
			if old, exists := h.clients[client.userID]; exists {
				close(old.send)
			}
			h.clients[client.userID] = client

		case client := <-h.unregister:
			if existing, ok := h.clients[client.userID]; ok && existing == client {
				delete(h.clients, client.userID)
				close(client.send)
			}

		case msg := <-h.broadcast:
			h.handleBroadcast(ctx, msg)

		case p := <-h.push:
			h.deliver(p.userID, p.payload)
		}
	}
}

func (h *Hub) NotifyDMCreated(recipientID string) {
	payload, err := json.Marshal(outgoingEvent{Type: msgTypeDMCreated})
	if err != nil {
		slog.Error("ws: marshal dm_created failed", "err", err)
		return
	}
	select {
	case h.push <- pushMsg{userID: recipientID, payload: payload}:
	default:
		slog.Warn("ws: push buffer full, dropping",
			"user", recipientID, "event", msgTypeDMCreated)
	}
}

func (h *Hub) shutdown() {
	for _, c := range h.clients {
		close(c.send)
	}
	h.clients = nil
}

func (h *Hub) handleBroadcast(parentCtx context.Context, msg IncomingMessage) {
	var in incomingWSMessage
	if err := json.Unmarshal(msg.data, &in); err != nil {
		slog.Warn("ws: invalid json from client", "user", msg.client.userID, "err", err)
		return
	}

	if strings.TrimSpace(in.Content) == "" || in.TargetID == "" {
		return
	}

	switch in.Type {
	case msgTypeChannel:
		h.handleChannelMessage(parentCtx, msg.client, in)
	case msgTypeDM:
		h.handleDMMessage(parentCtx, msg.client, in)
	default:
		slog.Warn("ws: unknown message type", "user", msg.client.userID, "type", in.Type)
	}
}

func (h *Hub) handleChannelMessage(parentCtx context.Context, sender *Client, in incomingWSMessage) {
	ctx, cancel := context.WithTimeout(parentCtx, dbTimeout)
	defer cancel()

	isMember, err := h.channelRepo.IsMember(ctx, in.TargetID, sender.userID)
	if err != nil {
		slog.Error("ws: IsMember failed", "user", sender.userID, "channel", in.TargetID, "err", err)
		return
	}
	if !isMember {
		slog.Warn("ws: non-member tried to post to channel",
			"user", sender.userID, "channel", in.TargetID)
		return
	}

	channelID := in.TargetID
	stored, err := h.messageRepo.Create(ctx, domain.Message{
		UserID:    sender.userID,
		Content:   in.Content,
		ChannelID: &channelID,
	})
	if err != nil {
		slog.Error("ws: persist channel message failed",
			"user", sender.userID, "channel", channelID, "err", err)
		return
	}

	out := outgoingWSMessage{
		ID:         stored.ID,
		Type:       msgTypeChannel,
		TargetType: "channel",
		TargetID:   channelID,
		Content:    stored.Content,
		SenderID:   sender.userID,
		Username:   sender.username,
		CreatedAt:  stored.CreatedAt,
	}
	payload, err := json.Marshal(out)
	if err != nil {
		slog.Error("ws: marshal outgoing failed", "err", err)
		return
	}

	members, err := h.channelRepo.GetMembers(ctx, channelID)
	if err != nil {
		slog.Error("ws: GetMembers failed", "channel", channelID, "err", err)
		return
	}

	for _, member := range members {
		if member.ID == sender.userID {
			continue
		}
		h.deliver(member.ID, payload)
	}
}

func (h *Hub) handleDMMessage(parentCtx context.Context, sender *Client, in incomingWSMessage) {
	ctx, cancel := context.WithTimeout(parentCtx, dbTimeout)
	defer cancel()

	chat, err := h.directRepo.FindByID(ctx, in.TargetID)
	if err != nil {
		if errors.Is(err, domain.ErrDirectChatNotFound) {
			slog.Warn("ws: DM to non-existent chat", "user", sender.userID, "chat", in.TargetID)
			return
		}
		slog.Error("ws: FindByID failed", "user", sender.userID, "chat", in.TargetID, "err", err)
		return
	}

	var recipientID string
	switch sender.userID {
	case chat.UserID1:
		recipientID = chat.UserID2
	case chat.UserID2:
		recipientID = chat.UserID1
	default:
		slog.Warn("ws: non-participant tried to post to DM",
			"user", sender.userID, "chat", chat.ID)
		return
	}

	chatID := chat.ID
	stored, err := h.messageRepo.Create(ctx, domain.Message{
		UserID:       sender.userID,
		Content:      in.Content,
		DirectChatID: &chatID,
	})
	if err != nil {
		slog.Error("ws: persist DM failed", "user", sender.userID, "chat", chatID, "err", err)
		return
	}

	out := outgoingWSMessage{
		ID:         stored.ID,
		Type:       msgTypeDM,
		TargetType: "direct",
		TargetID:   chatID,
		Content:    stored.Content,
		SenderID:   sender.userID,
		Username:   sender.username,
		CreatedAt:  stored.CreatedAt,
	}
	payload, err := json.Marshal(out)
	if err != nil {
		slog.Error("ws: marshal outgoing failed", "err", err)
		return
	}

	h.deliver(recipientID, payload)
}

func (h *Hub) deliver(userID string, payload []byte) {
	client, online := h.clients[userID]
	if !online {
		return
	}
	select {
	case client.send <- payload:
	default:
		slog.Warn("ws: client send buffer full, dropping connection", "user", userID)
		close(client.send)
		delete(h.clients, userID)
	}
}
