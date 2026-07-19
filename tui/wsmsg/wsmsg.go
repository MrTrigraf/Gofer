package wsmsg

import (
	"time"

	"github.com/gofer/tui/ws"
)

type IncomingMsg struct {
	Message ws.Incoming
}

type DisconnectedMsg struct{}

type DMCreatedMsg struct{}

type AckMsg struct {
	ClientMsgID string
	ID          string
	CreatedAt   time.Time
}

type AckTimeoutMsg struct {
	ClientMsgID string
	Attempt     int
}
