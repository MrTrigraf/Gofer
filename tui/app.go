package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gofer/tui/api"
	"github.com/gofer/tui/auth"
	"github.com/gofer/tui/screen"
	"github.com/gofer/tui/serverstore"
	"github.com/gofer/tui/views"
	"github.com/gofer/tui/views/popup"
	"github.com/gofer/tui/ws"
)

type netlinkStatus int

const (
	netlinkUnknown netlinkStatus = iota
	netlinkOnline
	netlinkOffline
)

type Model struct {
	width          int
	height         int
	hitboxes       *[]screen.Hitbox
	apiClient      *api.Client
	auth           auth.AuthState
	current        screen.Screen
	netlink        netlinkStatus
	copiedTarget   string
	ws             *ws.Client
	refreshing     bool
	store          *serverstore.Store
	serverPopup    screen.Screen
	swallowNextKey bool
	pingGen        int
	serverStatus   map[string]popup.PingStatus
}

func New(apiClient *api.Client, store *serverstore.Store) Model {
	boxes := make([]screen.Hitbox, 0, 16)
	return Model{
		hitboxes:     &boxes,
		apiClient:    apiClient,
		store:        store,
		current:      views.NewLogin(apiClient),
		netlink:      netlinkUnknown,
		serverStatus: make(map[string]popup.PingStatus),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.current.Init(),
		pingCmd(m.apiClient),
	)
}
