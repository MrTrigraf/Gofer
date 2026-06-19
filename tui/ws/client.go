package ws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Outgoing struct {
	Type     string `json:"type"`
	TargetID string `json:"target_id"`
	Content  string `json:"content"`
}

type Incoming struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	TargetType string    `json:"target_type"`
	TargetID   string    `json:"target_id"`
	Content    string    `json:"content"`
	SenderID   string    `json:"sender_id"`
	Username   string    `json:"username"`
	CreatedAt  time.Time `json:"created_at"`
}

type EventKind int

const (
	EventMessage EventKind = iota + 1
	EventDisconnected
	EventDMCreated
)

const msgTypeDMCreated = "dm_created"

type Event struct {
	Kind    EventKind
	Message Incoming
}

var (
	ErrClosed       = errors.New("ws client closed")
	ErrBufferFull   = errors.New("ws send buffer full")
	ErrUnauthorized = errors.New("ws unauthorized")
)

const (
	sendBufferSize   = 64
	eventsBufferSize = 64
	handshakeTimeout = 10 * time.Second
	writeWait        = 5 * time.Second
	pingPeriod       = 3 * time.Second
	pongWait         = 9 * time.Second
)

type Client struct {
	conn      *websocket.Conn
	sendCh    chan []byte
	events    chan Event
	done      chan struct{}
	closeOnce sync.Once
}

func Dial(ctx context.Context, url, token string) (*Client, error) {
	dialer := websocket.Dialer{
		HandshakeTimeout: handshakeTimeout,
	}
	header := http.Header{}
	header.Set("Authorization", "Bearer "+token)

	conn, resp, err := dialer.DialContext(ctx, url, header)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusUnauthorized {
			return nil, ErrUnauthorized
		}
		return nil, fmt.Errorf("ws dial: %w", err)
	}

	c := &Client{
		conn:   conn,
		sendCh: make(chan []byte, sendBufferSize),
		events: make(chan Event, eventsBufferSize),
		done:   make(chan struct{}),
	}
	go c.readLoop()
	go c.writeLoop()
	go c.pingLoop()
	return c, nil
}

func (c *Client) Events() <-chan Event {
	return c.events
}

func (c *Client) Send(msg Outgoing) error {
	select {
	case <-c.done:
		return ErrClosed
	default:
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal ws msg: %w", err)
	}

	select {
	case c.sendCh <- data:
		return nil
	case <-c.done:
		return ErrClosed
	default:
		return ErrBufferFull
	}
}

func (c *Client) Close() error {
	c.shutdown()
	return nil
}

func (c *Client) shutdown() {
	c.closeOnce.Do(func() {
		_ = c.conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(writeWait),
		)
		close(c.done)
		c.conn.Close()
	})
}

func (c *Client) readLoop() {
	defer func() {
		select {
		case c.events <- Event{Kind: EventDisconnected}:
		default:
		}
		c.shutdown()
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		c.conn.SetReadDeadline(time.Now().Add(pongWait))

		var in Incoming
		if err := json.Unmarshal(raw, &in); err != nil {
			continue
		}

		ev := Event{Kind: EventMessage, Message: in}
		if in.Type == msgTypeDMCreated {
			ev = Event{Kind: EventDMCreated}
		}

		select {
		case c.events <- ev:
		case <-c.done:
			return
		}
	}
}

func (c *Client) writeLoop() {
	for {
		select {
		case <-c.done:
			return
		case data := <-c.sendCh:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				c.shutdown()
				return
			}
		}
	}
}

func (c *Client) pingLoop() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			err := c.conn.WriteControl(
				websocket.PingMessage,
				[]byte{},
				time.Now().Add(writeWait),
			)
			if err != nil {
				c.shutdown()
				return
			}
		}
	}
}
