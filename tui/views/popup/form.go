package popup

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gofer/tui/screen"
	"github.com/gofer/tui/styles"
)

type FormResultMsg struct {
	Action    string
	Confirmed bool
	Value     string
}

const formWidth = 50

type formFocus int

const (
	formFocusInput formFocus = iota
	formFocusSubmit
	formFocusCancel
)

type FormModel struct {
	title       string
	label       string
	placeholder string
	submitBtn   string
	action      string

	input   textinput.Model
	focused formFocus

	parentW, parentH int
	originX, originY int

	hitboxes []screen.Hitbox
}

func NewForm(action, title, label, placeholder, submitLabel string, charLimit int) *FormModel {
	in := textinput.New()
	in.Placeholder = placeholder
	in.Prompt = ""
	in.Width = formWidth - 2 - 4 - 2
	if charLimit > 0 {
		in.CharLimit = charLimit
	}
	in.Focus()

	return &FormModel{
		title:       title,
		label:       label,
		placeholder: placeholder,
		submitBtn:   submitLabel,
		action:      action,
		input:       in,
		focused:     formFocusInput,
	}
}

// === screen.Screen ===

func (m *FormModel) Init() tea.Cmd             { return textinput.Blink }
func (m *FormModel) SetSize(w, h int)          { m.parentW, m.parentH = w, h }
func (m *FormModel) SetOrigin(x, y int)        { m.originX, m.originY = x, y }
func (m *FormModel) Hitboxes() []screen.Hitbox { return m.hitboxes }

func (m *FormModel) Update(msg tea.Msg) (screen.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, m.cancel()

		case "enter":
			if m.focused == formFocusCancel {
				return m, m.cancel()
			}
			return m, m.submit()

		case "tab":
			m.focusNext()
			m.applyFocus()
			return m, nil

		case "shift+tab":
			m.focusPrev()
			m.applyFocus()
			return m, nil

		case "left":
			if m.focused == formFocusInput {
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				return m, cmd
			}
			if m.focused == formFocusCancel {
				m.focused = formFocusSubmit
				m.applyFocus()
			}
			return m, nil

		case "right":
			if m.focused == formFocusInput {
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				return m, cmd
			}
			if m.focused == formFocusSubmit {
				m.focused = formFocusCancel
				m.applyFocus()
			}
			return m, nil
		}

	case tea.MouseMsg:
		if msg.Action != tea.MouseActionPress {
			return m, nil
		}
		id := screen.HitTest(m.hitboxes, msg.X, msg.Y)
		switch id {
		case "form_submit":
			return m, m.submit()
		case "form_cancel":
			return m, m.cancel()
		case "form_input":
			m.focused = formFocusInput
			m.applyFocus()
			return m, nil
		}
	}

	if m.focused == formFocusInput {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *FormModel) focusNext() {
	switch m.focused {
	case formFocusInput:
		m.focused = formFocusSubmit
	case formFocusSubmit:
		m.focused = formFocusCancel
	case formFocusCancel:
		m.focused = formFocusInput
	}
}

func (m *FormModel) focusPrev() {
	switch m.focused {
	case formFocusInput:
		m.focused = formFocusCancel
	case formFocusSubmit:
		m.focused = formFocusInput
	case formFocusCancel:
		m.focused = formFocusSubmit
	}
}

func (m *FormModel) applyFocus() {
	if m.focused == formFocusInput {
		m.input.Focus()
	} else {
		m.input.Blur()
	}
}

func (m *FormModel) submit() tea.Cmd {
	action := m.action
	value := m.input.Value()
	return func() tea.Msg {
		return FormResultMsg{Action: action, Confirmed: true, Value: value}
	}
}

func (m *FormModel) cancel() tea.Cmd {
	action := m.action
	return func() tea.Msg {
		return FormResultMsg{Action: action, Confirmed: false}
	}
}

// === VIEW ===

func (m *FormModel) View() string {
	m.hitboxes = m.hitboxes[:0]

	innerWidth := formWidth - 2 - 4

	center := lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center)

	title := center.Render(styles.StyleTitle.Render(m.title))
	label := styles.StyleFaint.Render(m.label)

	inputStyle := styles.StyleInput
	if m.focused == formFocusInput {
		inputStyle = styles.StyleInputFocused
	}
	fixedInput := lipgloss.NewStyle().Width(m.input.Width).Render(m.input.View())
	inputBox := inputStyle.Render(fixedInput)

	submitText := "[ " + m.submitBtn + " ]"
	cancelText := "[ Cancel ]"

	var submitLabel, cancelLabel string
	switch m.focused {
	case formFocusSubmit:
		submitLabel = styles.StyleAccent.Render(submitText)
		cancelLabel = styles.StyleItemInactive.Render(cancelText)
	case formFocusCancel:
		submitLabel = styles.StyleItemInactive.Render(submitText)
		cancelLabel = styles.StyleAccent.Render(cancelText)
	default:
		submitLabel = styles.StyleItemInactive.Render(submitText)
		cancelLabel = styles.StyleItemInactive.Render(cancelText)
	}

	gap := "   "
	gapW := lipgloss.Width(gap)
	submitW := lipgloss.Width(submitLabel)
	cancelW := lipgloss.Width(cancelLabel)
	rowW := submitW + gapW + cancelW
	leftPad := (innerWidth - rowW) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	rowPlain := submitLabel + gap + cancelLabel
	buttonsRow := lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center).Render(rowPlain)

	inner := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		label,
		inputBox,
		"",
		buttonsRow,
	)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorAccent).
		Padding(1, 2).
		Width(formWidth).
		Render(inner)

	placed := lipgloss.Place(
		m.parentW, m.parentH,
		lipgloss.Center, lipgloss.Center,
		box,
	)

	// === Hitbox'ы ===
	boxW := lipgloss.Width(box)
	boxH := lipgloss.Height(box)
	popupAbsX := m.originX + (m.parentW-boxW)/2
	popupAbsY := m.originY + (m.parentH-boxH)/2

	innerStartX := popupAbsX + 1 + 2 // border + padding-left
	innerStartY := popupAbsY + 1 + 1 // border + padding-top

	inputY := innerStartY + lipgloss.Height(title) + 1 + lipgloss.Height(label)
	inputClickableY := inputY + 1
	inputClickableX1 := innerStartX
	inputClickableX2 := inputClickableX1 + lipgloss.Width(inputBox) - 1
	m.hitboxes = append(m.hitboxes, screen.Hitbox{
		X1: inputClickableX1, Y1: inputClickableY,
		X2: inputClickableX2, Y2: inputClickableY,
		ID: "form_input",
	})

	// Y кнопок = после inputBox + blank.
	btnY := inputY + lipgloss.Height(inputBox) + 1

	submitX1 := innerStartX + leftPad
	submitX2 := submitX1 + submitW - 1
	cancelX1 := submitX2 + 1 + gapW
	cancelX2 := cancelX1 + cancelW - 1

	m.hitboxes = append(m.hitboxes,
		screen.Hitbox{X1: submitX1, Y1: btnY, X2: submitX2, Y2: btnY, ID: "form_submit"},
		screen.Hitbox{X1: cancelX1, Y1: btnY, X2: cancelX2, Y2: btnY, ID: "form_cancel"},
	)

	return placed
}
