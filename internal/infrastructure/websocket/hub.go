package websocket

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gofer/internal/usecase"
)

type Hub struct {
	clients     map[string]*Client
	broadcast   chan IncomingMessage
	register    chan *Client
	unregister  chan *Client
	channelRepo usecase.ChannelRepository
}

type WSMessage struct {
	Type       string    `json:"type"`
	TargetType string    `json:"target_type"`
	TargetID   string    `json:"target_id"`
	Content    string    `json:"content"`
	SenderID   string    `json:"sender_id"`
	Username   string    `json:"username"`
	CreatedAt  time.Time `json:"created_at"`
}

func NewHub(channelRepo usecase.ChannelRepository) *Hub {
	return &Hub{
		clients:     make(map[string]*Client),
		broadcast:   make(chan IncomingMessage, 256),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		channelRepo: channelRepo,
	}
}

func (h *Hub) Run() {
	for {
		select {

		case client := <-h.register:
			h.clients[client.userID] = client

		case client := <-h.unregister:
			if _, ok := h.clients[client.userID]; ok {
				delete(h.clients, client.userID)
				close(client.send)
			}
		case msg := <-h.broadcast:
			var wsMsg WSMessage
			if err := json.Unmarshal(msg.data, &wsMsg); err != nil {
				continue
			}

			switch wsMsg.Type {
			case "channel_message":
				members, err := h.channelRepo.GetMembers(context.Background(), wsMsg.TargetID)
				if err != nil {
					continue
				}
				for _, member := range members {
					client, online := h.clients[member.ID]
					if online && member.ID != msg.client.userID {
						client.send <- msg.data
					}
				}

			case "dm_message":
				recipient, online := h.clients[wsMsg.TargetID]
				if online {
					recipient.send <- msg.data
				}
			}
		}
	}
}

func (h *Hub) ClientsCount() int {
	return len(h.clients)
}
