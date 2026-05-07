package wsmsg

import (
	"github.com/gofer/tui/ws"
)

type IncomingMsg struct {
	Message ws.Incoming
}

type DisconnectedMsg struct{}
