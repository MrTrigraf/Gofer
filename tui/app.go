package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gofer/tui/api"
	"github.com/gofer/tui/auth"
	"github.com/gofer/tui/screen"
	"github.com/gofer/tui/views"
	"github.com/gofer/tui/ws"
)

type netlinkStatus int

const (
	netlinkUnknown netlinkStatus = iota
	netlinkOnline
	netlinkOffline
)

type Model struct {
	width        int
	height       int
	hitboxes     *[]screen.Hitbox
	apiClient    *api.Client
	auth         auth.AuthState
	current      screen.Screen
	netlink      netlinkStatus
	copiedTarget string
	ws           *ws.Client
}

func New(apiClient *api.Client) Model {
	boxes := make([]screen.Hitbox, 0, 16)
	return Model{
		hitboxes:  &boxes,
		apiClient: apiClient,
		current:   views.NewLogin(apiClient),
		netlink:   netlinkUnknown,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.current.Init(),
		pingCmd(m.apiClient),
	)
}
