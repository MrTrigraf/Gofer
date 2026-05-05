package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gofer/tui/api"
	"github.com/gofer/tui/auth"
	"github.com/gofer/tui/screen"
	"github.com/gofer/tui/views"
)

// netlinkStatus — состояние соединения с сервером.
type netlinkStatus int

const (
	netlinkUnknown netlinkStatus = iota // пока не пинговали
	netlinkOnline
	netlinkOffline
)

// Model — главная модель приложения, диспетчер всех экранов.
type Model struct {
	width    int
	height   int
	hitboxes *[]screen.Hitbox

	apiClient *api.Client

	// auth — состояние авторизации. Пустое (UserID=="") до успешного login.
	auth auth.AuthState

	current screen.Screen

	// netlink — состояние соединения, обновляется периодическим пингом.
	netlink      netlinkStatus
	copiedTarget string
}

// New принимает api-клиент извне (DI).
func New(apiClient *api.Client) Model {
	boxes := make([]screen.Hitbox, 0, 16)
	return Model{
		hitboxes:  &boxes,
		apiClient: apiClient,
		current:   views.NewLogin(apiClient),
		netlink:   netlinkUnknown,
	}
}

// Init запускает первый пинг сразу (чтобы не ждать 5 секунд до первого цикла).
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.current.Init(),
		pingCmd(m.apiClient),
	)
}
