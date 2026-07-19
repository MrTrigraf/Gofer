package views

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gofer/tui/api"
	"github.com/gofer/tui/auth"
	"github.com/gofer/tui/styles"
	"github.com/gofer/tui/ws"
	"github.com/gofer/tui/wsmsg"
	"github.com/google/uuid"
)

const (
	ChatTargetChannel = "channel"
	ChatTargetDirect  = "direct"
)

const historyLoadLimit = 50
const inputCharLimit = 2000
const minContentWidth = 8
const maxInputHeight = 3
const inputPrefixWidth = 2
const ackTimeout = 10 * time.Second

type msgStatus int

const (
	statusSent    msgStatus = iota // подтверждено сервером (история, входящие, ack)
	statusPending                  // отправлено локально, ждём ack
	statusFailed                   // доставка провалилась
)

type chatMessage struct {
	api.Message
	localID     int // трек-номер нашего исходящего; 0 = чужое/историческое
	clientMsgID string
	status      msgStatus // статус доставки
	sendAttempt int       // номер попытки отправки; матчит AckTimeoutMsg
}

type WSSendMsg struct {
	Type        string
	TargetID    string
	Content     string
	LocalID     int
	ClientMsgID string
}

// WSSendOKMsg — Send() успешно положил сообщение в WS-буфер.
// Статус sent ставит только AckMsg (сервер подтвердил персист).
type WSSendOKMsg struct {
	LocalID int
}

// WSSendFailedMsg — отправка не удалась (ws == nil или ошибка Send()).
type WSSendFailedMsg struct {
	LocalID int
}

type ChatPanel struct {
	api  *api.Client
	auth auth.AuthState

	targetType  string
	targetID    string
	displayName string

	messages []chatMessage
	loading  bool
	loadErr  error

	reqSeq    int
	activeReq int

	localSeq int

	input textarea.Model

	vp            viewport.Model
	stickToBottom bool
}

func NewChatPanel(client *api.Client, state auth.AuthState) *ChatPanel {
	in := textarea.New()
	in.Placeholder = "Type a message, Enter to send (Alt+Enter — newline)"
	in.CharLimit = inputCharLimit
	in.ShowLineNumbers = false
	in.FocusedStyle.CursorLine = lipgloss.NewStyle()
	in.BlurredStyle.CursorLine = lipgloss.NewStyle()
	in.FocusedStyle.Text = styles.StyleMessageText
	in.BlurredStyle.Text = styles.StyleMessageText
	in.FocusedStyle.Placeholder = styles.StyleFaint
	in.BlurredStyle.Placeholder = styles.StyleFaint
	in.Prompt = "" // свой prefix "> " рисуем в renderInput
	in.SetHeight(maxInputHeight)
	in.KeyMap.LinePrevious = key.NewBinding(
		key.WithKeys("ctrl+up", "ctrl+p"),
		key.WithHelp("ctrl+↑", "line up"),
	)
	in.KeyMap.LineNext = key.NewBinding(
		key.WithKeys("ctrl+down", "ctrl+n"),
		key.WithHelp("ctrl+↓", "line down"),
	)
	in.KeyMap.CharacterBackward = key.NewBinding(
		key.WithKeys("ctrl+left", "ctrl+b"),
		key.WithHelp("ctrl+←", "character backward"),
	)
	in.KeyMap.CharacterForward = key.NewBinding(
		key.WithKeys("ctrl+right", "ctrl+f"),
		key.WithHelp("ctrl+→", "character forward"),
	)
	in.KeyMap.InsertNewline = key.NewBinding(
		key.WithKeys("alt+enter", "shift+enter", "ctrl+j"),
		key.WithHelp("alt+enter", "insert newline"),
	)
	return &ChatPanel{
		api:           client,
		auth:          state,
		input:         in,
		vp:            viewport.New(0, 0),
		stickToBottom: true,
	}
}

func (p *ChatPanel) HasTarget() bool {
	return p.targetType != ""
}

func (p *ChatPanel) SetTarget(targetType, targetID, displayName string) tea.Cmd {
	if p.targetType == targetType && p.targetID == targetID {
		return nil
	}

	p.targetType = targetType
	p.targetID = targetID
	p.displayName = displayName
	p.messages = nil
	p.loadErr = nil
	p.loading = true

	p.input.SetValue("")
	p.input.Focus()

	p.stickToBottom = true

	p.reqSeq++
	p.activeReq = p.reqSeq
	reqID := p.activeReq

	return loadHistoryCmd(p.api, targetType, targetID, reqID)
}

func (p *ChatPanel) Clear() {
	p.targetType = ""
	p.targetID = ""
	p.displayName = ""
	p.messages = nil
	p.loadErr = nil
	p.loading = false
	p.input.SetValue("")
	p.input.Blur()
	p.reqSeq++
	p.activeReq = p.reqSeq
	p.stickToBottom = true
}

func (p *ChatPanel) Update(msg tea.Msg) (*ChatPanel, tea.Cmd) {
	switch m := msg.(type) {
	case historyLoadedMsg:
		if m.reqID != p.activeReq {
			return p, nil
		}
		p.loading = false
		p.loadErr = nil
		p.messages = p.messages[:0]
		for _, am := range m.messages {
			p.messages = append(p.messages, chatMessage{
				Message: am,
				status:  statusSent,
			})
		}
		return p, nil

	case historyFailedMsg:
		if m.reqID != p.activeReq {
			return p, nil
		}
		p.loading = false
		p.loadErr = m.err
		return p, nil

	case wsmsg.IncomingMsg:
		if !p.matchesIncoming(m.Message) {
			return p, nil
		}
		p.messages = append(p.messages, chatMessage{
			Message: api.Message{
				ID:        m.Message.ID,
				SenderID:  m.Message.SenderID,
				Username:  m.Message.Username,
				Content:   m.Message.Content,
				CreatedAt: m.Message.CreatedAt,
			},
			status: statusSent,
		})
		return p, nil

	case wsmsg.AckMsg:
		for i := range p.messages {
			if p.messages[i].clientMsgID == m.ClientMsgID && p.messages[i].clientMsgID != "" {
				p.messages[i].ID = m.ID
				p.messages[i].CreatedAt = m.CreatedAt
				p.messages[i].status = statusSent
				break
			}
		}
		return p, nil

	case wsmsg.AckTimeoutMsg:
		for i := range p.messages {
			cm := &p.messages[i]
			if cm.clientMsgID != m.ClientMsgID {
				continue
			}
			if cm.status == statusPending && cm.sendAttempt == m.Attempt {
				cm.status = statusFailed
			}
			break
		}
		return p, nil

	case WSSendOKMsg:
		return p, nil

	case WSSendFailedMsg:
		for i := range p.messages {
			if p.messages[i].localID == m.LocalID {
				p.messages[i].status = statusFailed
				break
			}
		}
		return p, nil

	case wsmsg.DisconnectedMsg:
		for i := range p.messages {
			cm := &p.messages[i]
			if cm.localID == 0 {
				continue // чужое или историческое
			}
			if cm.status == statusPending {
				cm.status = statusFailed
			}
		}
		return p, nil

	case tea.KeyMsg:
		if p.input.Focused() {
			switch {
			case m.Type == tea.KeyEnter && !m.Alt:
				return p, p.handleSend()
			case m.String() == "ctrl+r":
				return p, p.retryFailed()
			case m.Type == tea.KeyPgUp, m.Type == tea.KeyPgDown:
				var cmd tea.Cmd
				p.vp, cmd = p.vp.Update(msg)
				p.stickToBottom = p.vp.AtBottom()
				return p, cmd
			}
		}

	case tea.MouseMsg:
		if m.Button == tea.MouseButtonWheelUp || m.Button == tea.MouseButtonWheelDown {
			var cmd tea.Cmd
			p.vp, cmd = p.vp.Update(msg)
			p.stickToBottom = p.vp.AtBottom()
			return p, cmd
		}
	}
	if p.input.Focused() {
		var cmd tea.Cmd
		p.input, cmd = p.input.Update(msg)
		return p, cmd
	}
	return p, nil
}

func (p *ChatPanel) wsType() string {
	if p.targetType == ChatTargetDirect {
		return "dm_message"
	}
	return "channel_message"
}

func ackTimeoutCmd(clientMsgID string, attempt int) tea.Cmd {
	return tea.Tick(ackTimeout, func(time.Time) tea.Msg {
		return wsmsg.AckTimeoutMsg{ClientMsgID: clientMsgID, Attempt: attempt}
	})
}

func (p *ChatPanel) handleSend() tea.Cmd {
	content := strings.TrimSpace(p.input.Value())
	if content == "" {
		return nil
	}
	if !p.HasTarget() {
		return nil
	}

	p.localSeq++
	localID := p.localSeq
	clientMsgID := uuid.NewString()

	const firstAttempt = 1
	p.messages = append(p.messages, chatMessage{
		Message: api.Message{
			SenderID:  p.auth.UserID,
			Username:  p.auth.Username,
			Content:   content,
			CreatedAt: time.Now(),
		},
		localID:     localID,
		clientMsgID: clientMsgID,
		status:      statusPending,
		sendAttempt: firstAttempt,
	})

	p.input.SetValue("")

	out := WSSendMsg{
		Type:        p.wsType(),
		TargetID:    p.targetID,
		Content:     content,
		LocalID:     localID,
		ClientMsgID: clientMsgID,
	}
	return tea.Batch(
		func() tea.Msg { return out },
		ackTimeoutCmd(clientMsgID, firstAttempt),
	)
}

func (p *ChatPanel) retryFailed() tea.Cmd {
	var cmds []tea.Cmd
	for i := range p.messages {
		cm := &p.messages[i]
		if cm.status != statusFailed {
			continue
		}
		cm.status = statusPending
		cm.sendAttempt++

		out := WSSendMsg{
			Type:        p.wsType(),
			TargetID:    p.targetID,
			Content:     cm.Content,
			LocalID:     cm.localID,
			ClientMsgID: cm.clientMsgID,
		}
		attempt := cm.sendAttempt
		cmds = append(cmds,
			func() tea.Msg { return out },
			ackTimeoutCmd(cm.clientMsgID, attempt),
		)
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Sequence(cmds...)
}

func (p *ChatPanel) View(width, height int) string {
	if !p.HasTarget() {
		return p.renderEmpty(width, height)
	}

	if iw := width - inputPrefixWidth; iw > 0 {
		p.input.SetWidth(iw)
	}

	const headerH = 1
	inputH := p.inputHeight()

	bodyH := height - headerH - inputH
	if bodyH < 1 {
		bodyH = 1
	}

	header := p.renderHeader(width)
	body := p.renderBody(width, bodyH)
	input := p.renderInput(width)

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(lipgloss.JoinVertical(lipgloss.Left, header, body, input))
}

func (p *ChatPanel) renderEmpty(width, height int) string {
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center,
		styles.StyleFaint.Render("(no chat selected)"))
}

func (p *ChatPanel) renderHeader(width int) string {
	return styles.StyleAccent.Render(p.displayName)
}

func (p *ChatPanel) renderInput(width int) string {
	prefix := styles.StyleFaint.Render("> ")
	field := lipgloss.NewStyle().
		MaxHeight(p.inputHeight()).
		Render(p.input.View())

	return lipgloss.JoinHorizontal(lipgloss.Bottom, prefix, field)
}

func (p *ChatPanel) inputHeight() int {
	w := p.input.Width()
	if w < 1 {
		return 1
	}

	lines := 0
	for _, logical := range strings.Split(p.input.Value(), "\n") {
		lines += visualRows(logical, w)
	}

	if lines < 1 {
		lines = 1
	}
	if lines > maxInputHeight {
		lines = maxInputHeight
	}
	return lines
}

// visualRows — сколько экранных рядов займёт одна логическая строка при
// заданной ширине. Повторяет word-wrap из textarea: слово переносится
// целиком, если не влезает; слово длиннее строки рвётся по ширине.
func visualRows(s string, width int) int {
	if s == "" {
		return 1
	}

	rows := 1
	cur := 0

	for _, word := range strings.SplitAfter(s, " ") {
		wl := lipgloss.Width(word)

		// Слово само по себе длиннее строки — рвём по ширине.
		if wl > width {
			if cur > 0 {
				rows++
				cur = 0
			}
			rows += (wl - 1) / width
			cur = wl % width
			continue
		}

		if cur+wl > width {
			rows++
			cur = wl
		} else {
			cur += wl
		}
	}

	return rows
}

func (p *ChatPanel) renderBody(width, height int) string {
	switch {
	case p.loading:
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center,
			styles.StyleFaint.Render("loading..."))

	case p.loadErr != nil:
		errLine := styles.StyleDanger.Render("⚠ " + chatErrText(p.loadErr))
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, errLine)

	case len(p.messages) == 0:
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center,
			styles.StyleFaint.Render("no messages yet — be the first to write"))

	default:
		return p.renderMessages(width, height)
	}
}

func (p *ChatPanel) renderMessages(width, height int) string {
	p.vp.Width = width
	p.vp.Height = height

	var b strings.Builder
	for i, msg := range p.messages {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(p.renderMessageLine(msg, width))
	}
	p.vp.SetContent(b.String())
	if p.stickToBottom {
		p.vp.GotoBottom()
	}

	return lipgloss.NewStyle().Height(height).Render(p.vp.View())
}

func (p *ChatPanel) renderMessageLine(m chatMessage, width int) string {
	timeStr := styles.StyleFaint.Render(m.CreatedAt.Format("15:04"))

	var nameStyle lipgloss.Style
	if m.SenderID == p.auth.UserID {
		nameStyle = styles.StyleMessageSenderSelf
	} else {
		nameStyle = styles.StyleMessageSenderOther
	}
	name := nameStyle.Render(m.Username)
	sep := styles.StyleFaint.Render(" | ")

	prefix := timeStr + "  " + name + sep

	// Доступная ширина под текст = ширина панели минус префикс.
	// lipgloss.Width меряет визуальные колонки, игнорируя ANSI-коды.
	contentW := width - lipgloss.Width(prefix)
	if contentW < minContentWidth {
		contentW = minContentWidth
	}

	var content string
	switch m.status {
	case statusPending:
		// серый текст + серое многоточие — "в пути"
		content = styles.StyleFaint.Render(m.Content + " …")

	case statusFailed:
		// серый текст + красный крестик — глаз цепляется за ×,
		// но текст остаётся читаемым
		content = styles.StyleFaint.Render(m.Content) + styles.StyleDanger.Render(" ×")

	default: // statusSent
		content = styles.StyleMessageText.Render(m.Content)
	}

	// Width() включает word-wrap: lipgloss сам переносит по словам.
	// JoinHorizontal выравнивает многострочный блок справа от префикса,
	// автоматически добавляя отступ на строках 2+.
	wrapped := lipgloss.NewStyle().Width(contentW).Render(content)
	return lipgloss.JoinHorizontal(lipgloss.Top, prefix, wrapped)
}

func chatErrText(err error) string {
	switch {
	case errors.Is(err, api.ErrUnreachable):
		return "server unreachable"
	case errors.Is(err, api.ErrForbidden):
		return "no access to this chat"
	case errors.Is(err, api.ErrNotFound):
		return "chat not found"
	default:
		return "failed to load history"
	}
}

type historyLoadedMsg struct {
	reqID    int
	messages []api.Message
}

type historyFailedMsg struct {
	reqID int
	err   error
}

func loadHistoryCmd(c *api.Client, targetType, targetID string, reqID int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var (
			msgs []api.Message
			err  error
		)
		switch targetType {
		case ChatTargetChannel:
			msgs, err = c.GetChannelMessages(ctx, targetID, historyLoadLimit, time.Time{})
		case ChatTargetDirect:
			msgs, err = c.GetDMMessages(ctx, targetID, historyLoadLimit, time.Time{})
		default:
			err = fmt.Errorf("unknown target type: %q", targetType)
		}
		if err != nil {
			return historyFailedMsg{reqID: reqID, err: err}
		}
		return historyLoadedMsg{reqID: reqID, messages: msgs}
	}
}

func (p *ChatPanel) matchesIncoming(in ws.Incoming) bool {
	if p.targetID == "" || p.targetType == "" {
		return false
	}
	return in.TargetType == p.targetType && in.TargetID == p.targetID
}
