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
)

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

	// FIX(8.4.1.a): WS-heartbeat на стороне клиента.
	// pingPeriod — как часто шлём Ping серверу.
	// pongWait — сколько ждём Pong, прежде чем считать соединение мёртвым.
	// pongWait кратен pingPeriod (3 периода) — запас на сетевой джиттер,
	// чтобы единичная задержка Pong не вызвала ложный обрыв.
	pingPeriod = 3 * time.Second
	pongWait   = 9 * time.Second
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
	go c.pingLoop() // FIX(8.4.1.a): запускаем heartbeat-горутину
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

	// FIX(8.4.1.a): вооружаем ReadMessage таймаутом.
	// Без read deadline ReadMessage висит вечно при "тихом" обрыве сети
	// (TCP не закрылся, данные не ходят). Теперь:
	//  - ставим дедлайн pongWait вперёд;
	//  - PongHandler двигает дедлайн при каждом ответе сервера на наш Ping;
	//  - если за pongWait не пришло ни Pong, ни сообщения — ReadMessage
	//    падает по таймауту, readLoop выходит, наверх летит EventDisconnected.
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
		// FIX(8.4.1.a): обычный трафик — тоже признак живости соединения,
		// продлеваем дедлайн после каждого успешно прочитанного сообщения.
		c.conn.SetReadDeadline(time.Now().Add(pongWait))

		var in Incoming
		if err := json.Unmarshal(raw, &in); err != nil {
			continue
		}
		select {
		case c.events <- Event{Kind: EventMessage, Message: in}:
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

// FIX(8.4.1.a): pingLoop — клиентский heartbeat.
// Раз в pingPeriod шлёт серверу control-кадр Ping. Сервер на него
// автоматически отвечает Pong (стандартное поведение gorilla), и PongHandler
// в readLoop продлевает read deadline. Если WriteControl провалился —
// соединение уже мертвое, инициируем shutdown немедленно.
//
// WriteControl безопасно вызывать конкурентно с WriteMessage из writeLoop —
// это явно разрешено gorilla/websocket для control-кадров.
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
