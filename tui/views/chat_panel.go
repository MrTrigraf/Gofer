package views

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gofer/tui/api"
	"github.com/gofer/tui/auth"
	"github.com/gofer/tui/styles"
	"github.com/gofer/tui/ws"
	"github.com/gofer/tui/wsmsg"
)

const (
	ChatTargetChannel = "channel"
	ChatTargetDirect  = "direct"
)

const historyLoadLimit = 50

const inputCharLimit = 2000

type WSSendMsg struct {
	Type     string
	TargetID string
	Content  string
}

type ChatPanel struct {
	api  *api.Client
	auth auth.AuthState

	targetType  string
	targetID    string
	displayName string

	messages []api.Message
	loading  bool
	loadErr  error

	reqSeq    int
	activeReq int

	input textinput.Model
}

func NewChatPanel(client *api.Client, state auth.AuthState) *ChatPanel {
	in := textinput.New()
	in.Placeholder = "Type a message and press Enter..."
	in.CharLimit = inputCharLimit
	return &ChatPanel{
		api:   client,
		auth:  state,
		input: in,
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
}

func (p *ChatPanel) Update(msg tea.Msg) (*ChatPanel, tea.Cmd) {
	switch m := msg.(type) {
	case historyLoadedMsg:
		if m.reqID != p.activeReq {
			return p, nil
		}
		p.loading = false
		p.loadErr = nil
		p.messages = m.messages
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
		p.messages = append(p.messages, api.Message{
			ID:        m.Message.ID,
			SenderID:  m.Message.SenderID,
			Username:  m.Message.Username,
			Content:   m.Message.Content,
			CreatedAt: m.Message.CreatedAt,
		})
		return p, nil

	case wsmsg.DisconnectedMsg:
		return p, nil

	case tea.KeyMsg:
		if m.Type == tea.KeyEnter && p.input.Focused() {
			return p, p.handleSend()
		}
	}
	if p.input.Focused() {
		var cmd tea.Cmd
		p.input, cmd = p.input.Update(msg)
		return p, cmd
	}
	return p, nil
}

func (p *ChatPanel) handleSend() tea.Cmd {
	content := strings.TrimSpace(p.input.Value())
	if content == "" {
		return nil
	}
	if !p.HasTarget() {
		return nil
	}

	p.messages = append(p.messages, api.Message{
		SenderID:  p.auth.UserID,
		Username:  p.auth.Username,
		Content:   content,
		CreatedAt: time.Now(),
	})

	p.input.SetValue("")

	wsType := "channel_message"
	if p.targetType == ChatTargetDirect {
		wsType = "dm_message"
	}

	out := WSSendMsg{
		Type:     wsType,
		TargetID: p.targetID,
		Content:  content,
	}
	return func() tea.Msg { return out }
}

func (p *ChatPanel) View(width, height int) string {
	if !p.HasTarget() {
		return p.renderEmpty(width, height)
	}
	if iw := width - 2; iw > 0 && p.input.Width != iw {
		p.input.Width = iw
	}

	const headerH = 1
	const inputH = 1
	bodyH := height - headerH - inputH
	if bodyH < 1 {
		bodyH = 1
	}

	header := p.renderHeader(width)
	body := p.renderBody(width, bodyH)
	input := p.renderInput(width)

	return lipgloss.JoinVertical(lipgloss.Left, header, body, input)
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
	return prefix + p.input.View()
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
	visible := p.messages
	if len(visible) > height {
		visible = visible[len(visible)-height:]
	}

	var b strings.Builder
	for i, msg := range visible {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(p.renderMessageLine(msg))
	}

	missing := height - len(visible)
	for i := 0; i < missing; i++ {
		b.WriteByte('\n')
	}
	return b.String()
}

func (p *ChatPanel) renderMessageLine(m api.Message) string {
	timeStr := styles.StyleFaint.Render(m.CreatedAt.Format("15:04"))

	var nameStyle lipgloss.Style
	if m.SenderID == p.auth.UserID {
		nameStyle = styles.StyleAccent
	} else {
		nameStyle = styles.StyleAccent
	}
	name := nameStyle.Render(m.Username)
	sep := styles.StyleFaint.Render(" | ")

	return fmt.Sprintf("%s  %s%s%s", timeStr, name, sep, m.Content)
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
