package tui

import (
	"context"
	"log/slog"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gofer/pkg/clipboard"
	"github.com/gofer/tui/auth"
	"github.com/gofer/tui/screen"
	"github.com/gofer/tui/views"
	"github.com/gofer/tui/ws"
	"github.com/gofer/tui/wsmsg"
)

type wsConnectedMsg struct {
	client *ws.Client
}

type wsDialFailedMsg struct {
	err error
}

// FIX(8.4.1.d): wsReconnectMsg — тик "пора пробовать переподключиться".
// Приходит после паузы wsReconnectDelay и запускает повторный dialWSCmd.
type wsReconnectMsg struct{}

// FIX(8.4.1.d): пауза между попытками переподключения WS.
// Фиксированный backoff: провал dial -> ждём -> пробуем снова.
// 3с — достаточно, чтобы не зашумлять лог, и достаточно быстро для юзера.
const wsReconnectDelay = 3 * time.Second

func dialWSCmd(url, token string) tea.Cmd {
	return func() tea.Msg {
		client, err := ws.Dial(context.Background(), url, token)
		if err != nil {
			return wsDialFailedMsg{err: err}
		}
		return wsConnectedMsg{client: client}
	}
}

// FIX(8.4.1.d): scheduleReconnect — "через wsReconnectDelay пришли
// wsReconnectMsg". По аналогии со scheduleNextPing в netlink.go.
func scheduleReconnect() tea.Cmd {
	return tea.Tick(wsReconnectDelay, func(time.Time) tea.Msg {
		return wsReconnectMsg{}
	})
}

func waitWSCmd(client *ws.Client) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-client.Events()
		if !ok {
			return wsmsg.DisconnectedMsg{}
		}
		switch ev.Kind {
		case ws.EventMessage:
			return wsmsg.IncomingMsg{Message: ev.Message}
		case ws.EventDisconnected:
			return wsmsg.DisconnectedMsg{}
		}
		return nil
	}
}

func closeWSCmd(client *ws.Client) tea.Cmd {
	return func() tea.Msg {
		if client != nil {
			_ = client.Close()
		}
		return nil
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case auth.AuthenticatedMsg:
		m.auth = msg.State
		m.apiClient.SetAuth(msg.State.AccessToken)
		m.current = views.NewHome(m.apiClient, m.auth)
		return m, tea.Batch(
			m.current.Init(),
			dialWSCmd(m.apiClient.WSURL(), msg.State.AccessToken),
		)

	case wsConnectedMsg:
		m.ws = msg.client
		slog.Info("WS connected") // TEST(8.4.1.a)
		return m, waitWSCmd(m.ws)

	// FIX(8.4.1.d): dial провалился (сервер ещё лежит) — не сдаёмся,
	// планируем следующую попытку через wsReconnectDelay.
	case wsDialFailedMsg:
		slog.Warn("WS dial failed, will retry", "err", msg.err)
		return m, scheduleReconnect()

	// FIX(8.4.1.d): тик реконнекта — пробуем дозвониться снова.
	// Токен берём из m.auth (логина нет — он был сохранён при
	// auth.AuthenticatedMsg).
	// TODO(8.4.x): при долгом простое access-токен (15 мин) протухает —
	// dial вернёт ErrUnauthorized и реконнект будет крутиться впустую.
	// Правильное решение — refresh-токен флоу, вне рамок 8.4.1.
	case wsReconnectMsg:
		slog.Info("WS reconnecting...")
		return m, dialWSCmd(m.apiClient.WSURL(), m.auth.AccessToken)

	case wsmsg.IncomingMsg:
		var cmd tea.Cmd
		m.current, cmd = m.current.Update(msg)
		return m, tea.Batch(cmd, waitWSCmd(m.ws))

	// FIX(8.4.1.d): обрыв WS — сообщение пробрасываем в m.current
	// (ChatPanel перекрасит сообщения в failed, вариант X) И запускаем
	// цикл реконнекта.
	case wsmsg.DisconnectedMsg:
		slog.Warn("WS disconnected") // TEST(8.4.1.a)
		m.ws = nil
		var cmd tea.Cmd
		m.current, cmd = m.current.Update(msg)
		return m, tea.Batch(cmd, scheduleReconnect())

	case views.WSSendMsg:
		if m.ws == nil {
			slog.Warn("WS send skipped: not connected", "local_id", msg.LocalID)
			return m, func() tea.Msg {
				return views.WSSendFailedMsg{LocalID: msg.LocalID}
			}
		}
		out := ws.Outgoing{
			Type:     msg.Type,
			TargetID: msg.TargetID,
			Content:  msg.Content,
		}
		if err := m.ws.Send(out); err != nil {
			slog.Warn("WS send failed", "local_id", msg.LocalID, "err", err)
			return m, func() tea.Msg {
				return views.WSSendFailedMsg{LocalID: msg.LocalID}
			}
		}
		return m, func() tea.Msg {
			return views.WSSendOKMsg{LocalID: msg.LocalID}
		}

	case netlinkPingMsg:
		return m, pingCmd(m.apiClient)

	case netlinkResultMsg:
		if msg.online {
			m.netlink = netlinkOnline
		} else {
			m.netlink = netlinkOffline
		}
		return m, scheduleNextPing()

	case clipboard.CopiedMsg:
		if msg.Target == "footer_uuid" {
			m.copiedTarget = msg.Target
			return m, clipboard.ClearAfterTimeout(msg.Target)
		}
		var cmd tea.Cmd
		m.current, cmd = m.current.Update(msg)
		return m, cmd

	case clipboard.CopyFailedMsg:
		if msg.Target == "footer_uuid" {
			m.copiedTarget = "fail:" + msg.Target
			return m, clipboard.ClearAfterTimeout(msg.Target)
		}
		var cmd tea.Cmd
		m.current, cmd = m.current.Update(msg)
		return m, cmd

	case clipboard.ClearCopiedMsg:
		if m.copiedTarget == msg.Target || m.copiedTarget == "fail:"+msg.Target {
			m.copiedTarget = ""
		}
		var cmd tea.Cmd
		m.current, cmd = m.current.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "ctrl+q":
			return m, tea.Sequence(closeWSCmd(m.ws), tea.Quit)
		}
		var cmd tea.Cmd
		m.current, cmd = m.current.Update(msg)
		return m, cmd

	case tea.MouseMsg:
		if msg.Action != tea.MouseActionPress {
			return m, nil
		}
		id := screen.HitTest(*m.hitboxes, msg.X, msg.Y)
		switch id {
		case "close":
			return m, tea.Sequence(closeWSCmd(m.ws), tea.Quit)
		case "footer_uuid":
			return m, clipboard.CopyCmd("footer_uuid", m.auth.UserID)
		}
		var cmd tea.Cmd
		m.current, cmd = m.current.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	m.current, cmd = m.current.Update(msg)
	return m, cmd
}
