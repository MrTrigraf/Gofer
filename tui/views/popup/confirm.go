package popup

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gofer/tui/screen"
	"github.com/gofer/tui/styles"
)

// ResultMsg — сообщение, которое попап шлёт наверх после закрытия.
type ResultMsg struct {
	Action    string
	Confirmed bool
}

const popupWidth = 50

// focus — какая кнопка сейчас выделена.
type focus int

const (
	focusConfirm focus = iota
	focusCancel
)

type ConfirmModel struct {
	title      string
	message    string
	confirmBtn string
	isDanger   bool
	oneButton  bool
	action     string

	focused focus

	parentW, parentH int
	originX, originY int

	hitboxes []screen.Hitbox
}

func NewConfirm(action, title, message, confirmLabel string, danger bool) *ConfirmModel {
	return &ConfirmModel{
		action:     action,
		title:      title,
		message:    message,
		confirmBtn: confirmLabel,
		isDanger:   danger,
		focused:    focusConfirm,
	}
}

func NewWarning(action, title, message string) *ConfirmModel {
	return &ConfirmModel{
		action:     action,
		title:      title,
		message:    message,
		confirmBtn: "OK",
		oneButton:  true,
		focused:    focusConfirm,
	}
}

// === screen.Screen ===

func (m *ConfirmModel) Init() tea.Cmd             { return nil }
func (m *ConfirmModel) SetSize(w, h int)          { m.parentW, m.parentH = w, h }
func (m *ConfirmModel) SetOrigin(x, y int)        { m.originX, m.originY = x, y }
func (m *ConfirmModel) Hitboxes() []screen.Hitbox { return m.hitboxes }

func (m *ConfirmModel) Update(msg tea.Msg) (screen.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, m.cancel()
		case "enter":
			if m.focused == focusCancel {
				return m, m.cancel()
			}
			return m, m.confirm()
		case "left":
			if !m.oneButton {
				m.focused = focusConfirm
			}
			return m, nil
		case "right":
			if !m.oneButton {
				m.focused = focusCancel
			}
			return m, nil
		}
	case tea.MouseMsg:
		if msg.Action != tea.MouseActionPress {
			return m, nil
		}
		id := screen.HitTest(m.hitboxes, msg.X, msg.Y)
		switch id {
		case "popup_confirm":
			return m, m.confirm()
		case "popup_cancel":
			return m, m.cancel()
		}
	}
	return m, nil
}

func (m *ConfirmModel) confirm() tea.Cmd {
	action := m.action
	return func() tea.Msg { return ResultMsg{Action: action, Confirmed: true} }
}

func (m *ConfirmModel) cancel() tea.Cmd {
	action := m.action
	return func() tea.Msg { return ResultMsg{Action: action, Confirmed: false} }
}

// === VIEW ===

func (m *ConfirmModel) View() string {
	m.hitboxes = m.hitboxes[:0]

	innerWidth := popupWidth - 2 - 4

	center := lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center)

	title := center.Render(styles.StyleTitle.Render(m.title))
	message := center.Render(styles.StyleItemInactive.Render(m.message))
	messageH := lipgloss.Height(message)

	buttonsRow, confirmLocal, cancelLocal := m.buildButtons(innerWidth)

	inner := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		message,
		"",
		buttonsRow,
	)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorAccent).
		Padding(1, 2).
		Width(popupWidth).
		Render(inner)

	placed := lipgloss.Place(
		m.parentW, m.parentH,
		lipgloss.Center, lipgloss.Center,
		box,
	)

	boxW := lipgloss.Width(box)
	boxH := lipgloss.Height(box)
	popupAbsX := m.originX + (m.parentW-boxW)/2
	popupAbsY := m.originY + (m.parentH-boxH)/2

	btnRowOffsetInBox := 1 + 1 + 1 + 1 + messageH + 1
	btnAbsY := popupAbsY + btnRowOffsetInBox

	m.appendButtonHitbox("popup_confirm", popupAbsX+1+2+confirmLocal.startX, btnAbsY, confirmLocal.width)
	if !m.oneButton {
		m.appendButtonHitbox("popup_cancel", popupAbsX+1+2+cancelLocal.startX, btnAbsY, cancelLocal.width)
	}

	return placed
}

type btnSpan struct {
	startX int
	width  int
}

func (m *ConfirmModel) buildButtons(innerWidth int) (string, btnSpan, btnSpan) {
	confirmText := "[ " + m.confirmBtn + " ]"
	cancelText := "[ Cancel ]"

	// Стиль зависит от фокуса: выделенная — accent, невыделенная — inactive.
	var confirmLabel, cancelLabel string
	if m.oneButton {
		confirmLabel = styles.StyleAccent.Render(confirmText)
	} else if m.focused == focusConfirm {
		confirmLabel = styles.StyleAccent.Render(confirmText)
		cancelLabel = styles.StyleItemInactive.Render(cancelText)
	} else {
		confirmLabel = styles.StyleItemInactive.Render(confirmText)
		cancelLabel = styles.StyleAccent.Render(cancelText)
	}

	confirmW := lipgloss.Width(confirmLabel)

	if m.oneButton {
		leftPad := (innerWidth - confirmW) / 2
		row := lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center).Render(confirmLabel)
		return row, btnSpan{startX: leftPad, width: confirmW}, btnSpan{}
	}

	cancelW := lipgloss.Width(cancelLabel)
	gap := "   "
	gapW := lipgloss.Width(gap)

	// Порядок: confirm (слева), cancel (справа).
	rowPlain := confirmLabel + gap + cancelLabel
	rowW := confirmW + gapW + cancelW

	leftPad := (innerWidth - rowW) / 2
	if leftPad < 0 {
		leftPad = 0
	}

	row := lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center).Render(rowPlain)

	confirmSpan := btnSpan{startX: leftPad, width: confirmW}
	cancelSpan := btnSpan{startX: leftPad + confirmW + gapW, width: cancelW}
	return row, confirmSpan, cancelSpan
}

func (m *ConfirmModel) appendButtonHitbox(id string, x, y, width int) {
	m.hitboxes = append(m.hitboxes, screen.Hitbox{
		X1: x, Y1: y,
		X2: x + width - 1, Y2: y,
		ID: id,
	})
}
