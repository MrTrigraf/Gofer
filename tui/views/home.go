package views

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gofer/pkg/clipboard"
	"github.com/gofer/tui/api"
	"github.com/gofer/tui/auth"
	"github.com/gofer/tui/screen"
)

type activeTab int

// Порядок: DIRECT первый (слева), CHANNELS второй (справа).
const (
	tabDirect activeTab = iota
	tabChannels
)

const (
	chatAreaListReserve = 13
	chatAreaListOffset  = 3
)

// === МОДЕЛЬ ===

type HomeModel struct {
	apiClient *api.Client
	state     auth.AuthState

	tab activeTab

	// Channels
	channels   []api.Channel
	loadErr    error
	loading    bool
	selectedID string

	// DMs
	dms          []api.DirectChat
	dmsLoadErr   error
	dmsLoading   bool
	selectedDMID string

	// cursor — индекс элемента ПОД КУРСОРОМ в активном списке.
	cursor int

	// addMode — true, когда юзер вошёл в режим управления.
	addMode bool

	// popup — активный модальный попап. nil = нет попапа.
	popup     screen.Screen
	addListVP viewport.Model
	// actionErr — последняя ошибка действия (leave/delete). Сбрасывается при успехе.
	actionErr    error
	copiedTarget string

	width, height    int
	originX, originY int

	hitboxes []screen.Hitbox
	chat     *ChatPanel
}

func NewHome(apiClient *api.Client, state auth.AuthState) *HomeModel {
	return &HomeModel{
		apiClient: apiClient,
		state:     state,
		tab:       tabDirect,
		addListVP: viewport.New(0, 0),
		chat:      NewChatPanel(apiClient, state),
	}
}

// === screen.Screen ===

func (m *HomeModel) Init() tea.Cmd {
	m.loading = true
	m.dmsLoading = true
	return tea.Batch(
		loadChannelsCmd(m.apiClient),
		loadDMsCmd(m.apiClient),
		clipboard.CheckCmd(),
	)
}

func (m *HomeModel) SetSize(width, height int) { m.width, m.height = width, height }
func (m *HomeModel) SetOrigin(x, y int)        { m.originX, m.originY = x, y }

func (m *HomeModel) Hitboxes() []screen.Hitbox {
	if m.popup != nil {
		return append(m.hitboxes, m.popup.Hitboxes()...)
	}
	return m.hitboxes
}

// === ХЕЛПЕРЫ СОСТОЯНИЯ ===

func (m *HomeModel) channelName(id string) string {
	for _, ch := range m.channels {
		if ch.ID == id {
			return ch.Name
		}
	}
	return ""
}

// findDM — DM по ID или nil.
func (m *HomeModel) findDM(id string) *api.DirectChat {
	if id == "" {
		return nil
	}
	for i := range m.dms {
		if m.dms[i].ID == id {
			return &m.dms[i]
		}
	}
	return nil
}

// channelExists — есть ли канал с таким ID в текущем списке.
func (m *HomeModel) channelExists(id string) bool {
	if id == "" {
		return false
	}
	return m.channelName(id) != ""
}

func (m *HomeModel) toggleTab() {
	if m.tab == tabChannels {
		m.tab = tabDirect
	} else {
		m.tab = tabChannels
	}
	m.cursor = 0
}

func (m *HomeModel) listLen() int {
	if m.tab == tabChannels {
		return len(m.channels)
	}
	return len(m.dms)
}

func (m *HomeModel) activateCursor() tea.Cmd {
	if m.tab == tabChannels {
		if m.cursor >= 0 && m.cursor < len(m.channels) {
			m.selectedID = m.channels[m.cursor].ID
			return m.syncChatTarget()
		}
		return nil
	}
	if m.cursor >= 0 && m.cursor < len(m.dms) {
		m.selectedDMID = m.dms[m.cursor].ID
		return m.syncChatTarget()
	}
	return nil
}

func (m *HomeModel) syncSelectionFromCursor() tea.Cmd {
	if !m.addMode {
		return nil
	}
	return m.activateCursor()
}

func (m *HomeModel) syncChatTarget() tea.Cmd {
	if m.tab == tabChannels && m.selectedID != "" {
		name := m.channelName(m.selectedID)
		if name == "" {
			name = "unknown"
		}
		return m.chat.SetTarget(ChatTargetChannel, m.selectedID, "# "+name)
	}
	if m.tab == tabDirect && m.selectedDMID != "" {
		var username string
		for _, dm := range m.dms {
			if dm.ID == m.selectedDMID {
				username = dm.OtherUsername
				break
			}
		}
		if username == "" {
			username = "unknown"
		}
		return m.chat.SetTarget(ChatTargetDirect, m.selectedDMID, "@ "+username)
	}
	m.chat.Clear()
	return nil
}

func (m *HomeModel) ensureCursorVisible() {
	if !m.addMode {
		return
	}
	h := m.addListVP.Height
	if h <= 0 {
		return
	}
	if m.cursor < m.addListVP.YOffset {
		m.addListVP.YOffset = m.cursor
	} else if m.cursor > m.addListVP.YOffset+h-1 {
		m.addListVP.YOffset = m.cursor - h + 1
	}
}
