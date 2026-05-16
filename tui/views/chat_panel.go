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

// FIX(8.4.1.b): окно "подозрительных" sent-сообщений.
// При обрыве WS все наши исходящие, ставшие sent за последние
// recentSentWindow, перекрашиваются в failed (вариант X): они могли
// уйти в мёртвый TCP-буфер и не дойти до сервера.
const recentSentWindow = 10 * time.Second

// FIX(8.4.1.b): статус доставки нашего исходящего сообщения.
type msgStatus int

const (
	statusSent    msgStatus = iota // на сервере (история, входящие, подтверждённые)
	statusPending                  // отправлено локально, ждём результата Send()
	statusFailed                   // доставка провалилась
)

// FIX(8.4.1.b): chatMessage — api.Message + UI-метаданные о доставке.
// api.Message не трогаем: это DTO транспортного слоя, статус — чисто
// UI-понятие, ему здесь не место. Обёртка живёт только внутри views.
type chatMessage struct {
	api.Message
	localID int       // трек-номер нашего исходящего; 0 = чужое/историческое
	status  msgStatus // статус доставки
	sentAt  time.Time // момент перехода в statusSent (для recentSentWindow)
}

type WSSendMsg struct {
	Type     string
	TargetID string
	Content  string
	LocalID  int // FIX(8.4.1.b): трек-номер для последующего OK/Failed
}

// FIX(8.4.1.b): результаты отправки, приходят из update.go.
// WSSendOKMsg — Send() успешно положил сообщение в WS-буфер.
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

	messages []chatMessage // FIX(8.4.1.b): было []api.Message
	loading  bool
	loadErr  error

	reqSeq    int
	activeReq int

	localSeq int // FIX(8.4.1.b): счётчик localID для исходящих

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
		// FIX(8.4.1.b): история — сообщения уже на сервере, статус sent.
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
		// FIX(8.4.1.b): входящее с сервера — статус sent.
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

	// FIX(8.4.1.b): Send() положил сообщение в WS-буфер — pending → sent.
	case WSSendOKMsg:
		for i := range p.messages {
			if p.messages[i].localID == m.LocalID {
				if p.messages[i].status == statusPending {
					p.messages[i].status = statusSent
					p.messages[i].sentAt = time.Now()
				}
				break
			}
		}
		return p, nil

	// FIX(8.4.1.b): отправка провалилась — помечаем сообщение failed.
	case WSSendFailedMsg:
		for i := range p.messages {
			if p.messages[i].localID == m.LocalID {
				p.messages[i].status = statusFailed
				break
			}
		}
		return p, nil

	// FIX(8.4.1.b): обрыв WS (вариант X). Все НАШИ исходящие
	// (localID != 0) под подозрением: pending — точно не дошли;
	// недавно-sent — могли уйти в мёртвый сокет. Перекрашиваем в failed.
	case wsmsg.DisconnectedMsg:
		now := time.Now()
		for i := range p.messages {
			cm := &p.messages[i]
			if cm.localID == 0 {
				continue // чужое или историческое — не трогаем
			}
			switch cm.status {
			case statusPending:
				cm.status = statusFailed
			case statusSent:
				if now.Sub(cm.sentAt) < recentSentWindow {
					cm.status = statusFailed
				}
			}
		}
		return p, nil

	case tea.KeyMsg:
		// FIX(8.4.1.e): Enter — отправка нового, Ctrl+R — ретрай failed.
		// Оба действия только при сфокусированном поле ввода — для
		// единообразия и чтобы ctrl+r не срабатывал вне чата.
		if p.input.Focused() {
			switch {
			case m.Type == tea.KeyEnter:
				return p, p.handleSend()
			case m.String() == "ctrl+r":
				return p, p.retryFailed()
			}
		}
	}
	if p.input.Focused() {
		var cmd tea.Cmd
		p.input, cmd = p.input.Update(msg)
		return p, cmd
	}
	return p, nil
}

// FIX(8.4.1.e): wsType возвращает тип WS-сообщения для текущего чата.
// Вынесено из handleSend, чтобы handleSend и retryFailed не дублировали
// одно и то же правило (пункт #5 TODO).
func (p *ChatPanel) wsType() string {
	if p.targetType == ChatTargetDirect {
		return "dm_message"
	}
	return "channel_message"
}

func (p *ChatPanel) handleSend() tea.Cmd {
	content := strings.TrimSpace(p.input.Value())
	if content == "" {
		return nil
	}
	if !p.HasTarget() {
		return nil
	}

	// FIX(8.4.1.b): выдаём трек-номер и кладём echo как pending.
	p.localSeq++
	localID := p.localSeq

	p.messages = append(p.messages, chatMessage{
		Message: api.Message{
			SenderID:  p.auth.UserID,
			Username:  p.auth.Username,
			Content:   content,
			CreatedAt: time.Now(),
		},
		localID: localID,
		status:  statusPending,
	})

	p.input.SetValue("")

	out := WSSendMsg{
		Type:     p.wsType(), // FIX(8.4.1.e): было инлайн-вычисление
		TargetID: p.targetID,
		Content:  content,
		LocalID:  localID, // FIX(8.4.1.b)
	}
	return func() tea.Msg { return out }
}

// FIX(8.4.1.e): retryFailed переотправляет ВСЕ failed-сообщения разом.
// Каждое возвращается в statusPending и переотправляется с тем же
// localID — новый не выдаём, иначе WSSendOKMsg/WSSendFailedMsg не найдут
// запись по localID. Порядок отправки через tea.Batch не гарантирован —
// на сервере сообщения могут перемешаться (принятое ограничение, вариант I).
func (p *ChatPanel) retryFailed() tea.Cmd {
	var cmds []tea.Cmd
	for i := range p.messages {
		cm := &p.messages[i]
		if cm.status != statusFailed {
			continue
		}
		cm.status = statusPending
		cm.sentAt = time.Time{} // pending его не читает, обнуляем для чистоты

		out := WSSendMsg{
			Type:     p.wsType(),
			TargetID: p.targetID,
			Content:  cm.Content,
			LocalID:  cm.localID,
		}
		cmds = append(cmds, func() tea.Msg { return out })
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
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

// FIX(8.4.1.b): рендер строки зависит от статуса доставки.
func (p *ChatPanel) renderMessageLine(m chatMessage) string {
	timeStr := styles.StyleFaint.Render(m.CreatedAt.Format("15:04"))

	var nameStyle lipgloss.Style
	if m.SenderID == p.auth.UserID {
		nameStyle = styles.StyleAccent
	} else {
		nameStyle = styles.StyleAccent
	}
	name := nameStyle.Render(m.Username)
	sep := styles.StyleFaint.Render(" | ")

	switch m.status {
	case statusPending:
		// серый текст + серое многоточие — "в пути"
		content := styles.StyleFaint.Render(m.Content + " …")
		return fmt.Sprintf("%s  %s%s%s", timeStr, name, sep, content)

	case statusFailed:
		// серый текст + красный крестик — глаз цепляется за ×,
		// но текст остаётся читаемым
		content := styles.StyleFaint.Render(m.Content)
		mark := styles.StyleDanger.Render(" ×")
		return fmt.Sprintf("%s  %s%s%s%s", timeStr, name, sep, content, mark)

	default: // statusSent
		return fmt.Sprintf("%s  %s%s%s", timeStr, name, sep, m.Content)
	}
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
