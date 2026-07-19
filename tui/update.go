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

type wsReconnectMsg struct{}

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
		case ws.EventDMCreated:
			return wsmsg.DMCreatedMsg{}
		case ws.EventAck:
			return wsmsg.AckMsg{
				ClientMsgID: ev.Message.ClientMsgID,
				ID:          ev.Message.ID,
				CreatedAt:   ev.Message.CreatedAt,
			}
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
		slog.Info("WS connected")
		return m, waitWSCmd(m.ws)

	case wsDialFailedMsg:
		slog.Warn("WS dial failed, will retry", "err", msg.err)
		return m, scheduleReconnect()

	case wsReconnectMsg:
		slog.Info("WS reconnecting...")
		return m, dialWSCmd(m.apiClient.WSURL(), m.auth.AccessToken)

	case wsmsg.IncomingMsg:
		var cmd tea.Cmd
		m.current, cmd = m.current.Update(msg)
		return m, tea.Batch(cmd, waitWSCmd(m.ws))

	case wsmsg.DMCreatedMsg:
		var cmd tea.Cmd
		m.current, cmd = m.current.Update(msg)
		return m, tea.Batch(cmd, waitWSCmd(m.ws))

	case wsmsg.AckMsg:
		var cmd tea.Cmd
		m.current, cmd = m.current.Update(msg)
		return m, tea.Batch(cmd, waitWSCmd(m.ws))

	case wsmsg.DisconnectedMsg:
		slog.Warn("WS disconnected")
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
			Type:        msg.Type,
			TargetID:    msg.TargetID,
			Content:     msg.Content,
			ClientMsgID: msg.ClientMsgID,
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
