package popup

import (
	"net/url"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gofer/tui/screen"
	"github.com/gofer/tui/styles"
)

type ServerConnectMsg struct{ Addr string }

type ServerAddMsg struct{ Addr string }

type ServerDeleteMsg struct{ Addr string }

type ServerCloseMsg struct{}

type PingStatus int

const (
	PingUnknown PingStatus = iota
	PingOnline
	PingOffline
)

const serverPopupWidth = 54

type zone int

const (
	zoneList zone = iota
	zoneButtons
)

type btn int

const (
	btnConnect btn = iota
	btnAdd
	btnDelete
	btnCount
)

type ServerModel struct {
	servers          []string
	selected         string
	status           map[string]PingStatus
	cursor           int
	zone             zone
	btn              btn
	addForm          *FormModel
	notice           string
	parentW, parentH int
	originX, originY int
	hitboxes         []screen.Hitbox
}

func NewServer(servers []string, selected string, status map[string]PingStatus) *ServerModel {
	if status == nil {
		status = make(map[string]PingStatus)
	}
	m := &ServerModel{
		servers:  servers,
		selected: selected,
		status:   status,
	}
	for i, addr := range servers {
		if addr == selected {
			m.cursor = i
			break
		}
	}
	if len(servers) == 0 {
		m.zone = zoneButtons
		m.btn = btnAdd
	} else {
		m.zone = zoneList
	}
	return m
}

// === screen.Screen ===
func (m *ServerModel) Init() tea.Cmd             { return nil }
func (m *ServerModel) SetSize(w, h int)          { m.parentW, m.parentH = w, h }
func (m *ServerModel) SetOrigin(x, y int)        { m.originX, m.originY = x, y }
func (m *ServerModel) Hitboxes() []screen.Hitbox { return m.hitboxes }

func (m *ServerModel) SetStatus(addr string, s PingStatus) {
	m.status[addr] = s
}

func (m *ServerModel) Update(msg tea.Msg) (screen.Screen, tea.Cmd) {
	if m.addForm != nil {
		if res, ok := msg.(FormResultMsg); ok {
			if res.Confirmed {
				return m, m.validateAndAdd(res.Value)
			}
			m.addForm = nil
			return m, nil
		}
		updated, cmd := m.addForm.Update(msg)
		if f, ok := updated.(*FormModel); ok {
			m.addForm = f
		}
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return ServerCloseMsg{} }

		case "tab", "shift+tab":
			m.notice = ""
			if len(m.servers) == 0 {
				m.zone = zoneButtons
				return m, nil
			}
			if m.zone == zoneList {
				m.zone = zoneButtons
			} else {
				m.zone = zoneList
			}
			return m, nil

		case "up":
			m.notice = ""
			if m.zone == zoneList && m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case "down":
			m.notice = ""
			if m.zone == zoneList && m.cursor < len(m.servers)-1 {
				m.cursor++
			}
			return m, nil

		case "left":
			if m.zone == zoneButtons {
				m.btn = (m.btn + btnCount - 1) % btnCount
			}
			return m, nil

		case "right":
			if m.zone == zoneButtons {
				m.btn = (m.btn + 1) % btnCount
			}
			return m, nil

		case "enter":
			if m.zone == zoneList {
				return m, m.connectCurrent()
			}
			switch m.btn {
			case btnConnect:
				return m, m.connectCurrent()
			case btnAdd:
				m.notice = ""
				m.openAddForm()
				return m, m.addForm.Init()
			case btnDelete:
				return m, m.deleteCurrent()
			}
			return m, nil

		case "a":
			m.notice = ""
			m.openAddForm()
			return m, m.addForm.Init()

		case "d":
			return m, m.deleteCurrent()
		}

	case tea.MouseMsg:
		if msg.Action != tea.MouseActionPress {
			return m, nil
		}
		id := screen.HitTest(m.hitboxes, msg.X, msg.Y)
		switch {
		case id == "srv_add":
			m.notice = ""
			m.zone = zoneButtons
			m.btn = btnAdd
			m.openAddForm()
			return m, m.addForm.Init()
		case id == "srv_delete":
			m.zone = zoneButtons
			m.btn = btnDelete
			return m, m.deleteCurrent()
		case id == "srv_connect":
			m.zone = zoneButtons
			m.btn = btnConnect
			return m, m.connectCurrent()
		case strings.HasPrefix(id, "srv_row_"):
			m.notice = ""
			idx := rowIndex(id)
			if idx >= 0 && idx < len(m.servers) {
				m.cursor = idx
				m.zone = zoneList
			}
			return m, nil
		}
	}
	return m, nil
}

func (m *ServerModel) connectCurrent() tea.Cmd {
	if len(m.servers) == 0 {
		m.notice = "No server to connect"
		return nil
	}
	if m.cursor < 0 || m.cursor >= len(m.servers) {
		return nil
	}
	addr := m.servers[m.cursor]
	return func() tea.Msg { return ServerConnectMsg{Addr: addr} }
}

func (m *ServerModel) deleteCurrent() tea.Cmd {
	if len(m.servers) == 0 {
		m.notice = "No servers to delete"
		return nil
	}
	if m.cursor < 0 || m.cursor >= len(m.servers) {
		return nil
	}
	addr := m.servers[m.cursor]
	return func() tea.Msg { return ServerDeleteMsg{Addr: addr} }
}

func (m *ServerModel) openAddForm() {
	m.addForm = NewForm(
		"add_server:-",
		"Add server",
		"Server address:",
		"host:port or http://host:port",
		"Add",
		200,
	)
}

func (m *ServerModel) validateAndAdd(raw string) tea.Cmd {
	addr := strings.TrimSpace(raw)
	if addr == "" {
		if m.addForm != nil {
			m.addForm.SetError("Address cannot be empty")
		}
		return nil
	}

	if !strings.Contains(addr, "://") {
		addr = "http://" + addr
	}

	u, err := url.Parse(addr)
	if err != nil || u.Host == "" ||
		(u.Scheme != "http" && u.Scheme != "https") {
		if m.addForm != nil {
			m.addForm.SetError("Invalid address — use host:port or http://host:port")
		}
		return nil
	}

	m.addForm = nil
	finalAddr := addr
	return func() tea.Msg { return ServerAddMsg{Addr: finalAddr} }
}

// === VIEW ===
func (m *ServerModel) View() string {
	m.hitboxes = m.hitboxes[:0]

	if m.addForm != nil {
		m.addForm.SetSize(m.parentW, m.parentH)
		m.addForm.SetOrigin(m.originX, m.originY)
		return m.addForm.View()
	}

	innerWidth := serverPopupWidth - 2 - 4

	title := lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center).
		Render(styles.StyleTitle.Render("Servers"))

	list := m.renderList(innerWidth)
	connectRow, addDeleteRow := m.renderButtons(innerWidth)

	noticeLine := " "
	if m.notice != "" {
		noticeLine = lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center).
			Render(styles.StyleDanger.Render(m.notice))
	}

	inner := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		list,
		"",
		connectRow,
		addDeleteRow,
		"",
		noticeLine,
	)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorAccent).
		Padding(1, 2).
		Width(serverPopupWidth).
		Render(inner)

	placed := lipgloss.Place(
		m.parentW, m.parentH,
		lipgloss.Center, lipgloss.Center,
		box,
	)

	m.registerHitboxes(box, innerWidth, title, list)
	return placed
}

// renderList рисует список серверов БЕЗ обёртки центрирования
// вокруг блока строк — раньше двойное центрирование (список + весь box)
// в сочетании с цветными иконками съедало левый край строки (иконку).
// Строки рисуем по левому краю; отступ задаёт паддинг box'а.
func (m *ServerModel) renderList(innerWidth int) string {
	if len(m.servers) == 0 {
		return lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center).
			Render(styles.StyleFaint.Render("no servers available"))
	}

	var lines []string
	for i, addr := range m.servers {
		cursor := "  "
		if i == m.cursor {
			if m.zone == zoneList {
				cursor = styles.StyleAccent.Render("▸ ")
			} else {
				cursor = styles.StyleItemInactive.Render("▸ ")
			}
		}

		icon := m.statusIcon(addr)

		label := addr
		if addr == m.selected {
			label = styles.StyleItemActive.Render(label)
		} else {
			label = styles.StyleItemInactive.Render(label)
		}

		lines = append(lines, cursor+icon+" "+label)
	}
	// Левое выравнивание, ширина фиксирована — строки не центрируются,
	// иконка в начале не обрезается.
	return lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Left).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m *ServerModel) statusIcon(addr string) string {
	// Индикатор статуса: один и тот же кружок ●, цвет несёт смысл.
	// Зелёный — доступен, красный — недоступен, серый — ждём ответа.
	// ● (U+25CF) выбран потому, что kitty рисует его шириной 1 (в отличие
	// от ✓/✗, которые получают эмодзи-ширину и стираются при диффе кадров).
	switch m.status[addr] {
	case PingOnline:
		return styles.StyleOK.Render("●")
	case PingOffline:
		return styles.StyleDanger.Render("●")
	default:
		return styles.StyleFaint.Render("●")
	}
}

func (m *ServerModel) btnStyle(which btn) lipgloss.Style {
	if m.zone == zoneButtons && m.btn == which {
		return styles.StyleAccent
	}
	return styles.StyleItemInactive
}

func (m *ServerModel) renderButtons(innerWidth int) (string, string) {
	conn := m.btnStyle(btnConnect).Render("[ Connect ]")
	connectRow := lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center).Render(conn)

	add := m.btnStyle(btnAdd).Render("[ Add ]")
	del := m.btnStyle(btnDelete).Render("[ Delete ]")
	addDeleteRow := lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center).
		Render(add + "   " + del)

	return connectRow, addDeleteRow
}

func (m *ServerModel) registerHitboxes(box string, innerWidth int, title, list string) {
	boxW := lipgloss.Width(box)
	boxH := lipgloss.Height(box)
	popupAbsX := m.originX + (m.parentW-boxW)/2
	popupAbsY := m.originY + (m.parentH-boxH)/2

	innerStartX := popupAbsX + 1 + 2
	innerStartY := popupAbsY + 1 + 1

	listStartY := innerStartY + lipgloss.Height(title) + 1

	if len(m.servers) > 0 {
		for i := range m.servers {
			rowY := listStartY + i
			m.hitboxes = append(m.hitboxes, screen.Hitbox{
				X1: innerStartX, Y1: rowY,
				X2: innerStartX + innerWidth - 1, Y2: rowY,
				ID: "srv_row_" + itoa(i),
			})
		}
	}

	connectY := listStartY + lipgloss.Height(list) + 1
	conn := styles.StyleAccent.Render("[ Connect ]")
	connW := lipgloss.Width(conn)
	connLeftPad := (innerWidth - connW) / 2
	if connLeftPad < 0 {
		connLeftPad = 0
	}
	connX1 := innerStartX + connLeftPad
	connX2 := connX1 + connW - 1
	m.hitboxes = append(m.hitboxes, screen.Hitbox{
		X1: connX1, Y1: connectY, X2: connX2, Y2: connectY, ID: "srv_connect",
	})

	adY := connectY + 1
	add := styles.StyleItemInactive.Render("[ Add ]")
	del := styles.StyleItemInactive.Render("[ Delete ]")
	gapW := 3
	addW := lipgloss.Width(add)
	delW := lipgloss.Width(del)
	rowW := addW + gapW + delW
	leftPad := (innerWidth - rowW) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	addX1 := innerStartX + leftPad
	addX2 := addX1 + addW - 1
	delX1 := addX2 + 1 + gapW
	delX2 := delX1 + delW - 1
	m.hitboxes = append(m.hitboxes,
		screen.Hitbox{X1: addX1, Y1: adY, X2: addX2, Y2: adY, ID: "srv_add"},
		screen.Hitbox{X1: delX1, Y1: adY, X2: delX2, Y2: adY, ID: "srv_delete"},
	)
}

func rowIndex(id string) int {
	s := strings.TrimPrefix(id, "srv_row_")
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return -1
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
