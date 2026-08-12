package tui

import (
	"context"
	"errors"
	"log/slog"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gofer/pkg/clipboard"
	"github.com/gofer/tui/api"
	"github.com/gofer/tui/auth"
	"github.com/gofer/tui/screen"
	"github.com/gofer/tui/views"
	"github.com/gofer/tui/views/popup"
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

type wsRefreshedMsg struct {
	token string
}

type wsAuthLostMsg struct{}

const wsReconnectDelay = 3 * time.Second
const serverPingInterval = 5 * time.Second
const serverPingTimeout = 5 * time.Second

type serverPingResultMsg struct {
	addr   string
	online bool
	gen    int
}

type serverPingTickMsg struct {
	gen int
}

func pingServerCmd(addr string, gen int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), serverPingTimeout)
		defer cancel()
		err := api.Ping(ctx, addr)
		return serverPingResultMsg{addr: addr, online: err == nil, gen: gen}
	}
}

func pingAllServersCmd(servers []string, gen int) tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(servers)+1)
	for _, addr := range servers {
		cmds = append(cmds, pingServerCmd(addr, gen))
	}
	cmds = append(cmds, scheduleServerPing(gen))
	return tea.Batch(cmds...)
}

func scheduleServerPing(gen int) tea.Cmd {
	return tea.Tick(serverPingInterval, func(time.Time) tea.Msg {
		return serverPingTickMsg{gen: gen}
	})
}

func dialWSCmd(url, token string) tea.Cmd {
	return func() tea.Msg {
		client, err := ws.Dial(context.Background(), url, token)
		if err != nil {
			return wsDialFailedMsg{err: err}
		}
		return wsConnectedMsg{client: client}
	}
}

func refreshTokenCmd(client *api.Client, refreshToken string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		pair, err := client.Refresh(ctx, refreshToken)
		if err != nil {
			slog.Warn("token refresh failed, session lost", "err", err)
			return wsAuthLostMsg{}
		}
		return wsRefreshedMsg{token: pair.AccessToken}
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
		case ws.EventDMDeleted:
			return wsmsg.DMDeletedMsg{}
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
	if m.serverPopup != nil {
		switch res := msg.(type) {
		case popup.ServerConnectMsg:
			m.serverPopup = nil
			m.swallowNextKey = true
			m.pingGen++
			if err := m.store.Select(res.Addr); err == nil {
				_ = m.store.Save()
			}
			m.apiClient.SetBaseURL(res.Addr)
			return m, pingCmd(m.apiClient)
		case popup.ServerAddMsg:
			_ = m.store.Add(res.Addr)
			_ = m.store.Save()
			m.serverPopup = m.newServerPopup()
			m.pingGen++
			return m, pingAllServersCmd(m.store.Servers, m.pingGen)
		case popup.ServerDeleteMsg:
			wasSelected := m.store.Selected == res.Addr
			m.store.Delete(res.Addr)
			_ = m.store.Save()
			delete(m.serverStatus, res.Addr) // адрес больше не нужен в карте статусов
			if wasSelected {
				m.apiClient.SetBaseURL("")
				m.netlink = netlinkUnknown
			}
			m.serverPopup = m.newServerPopup()
			m.pingGen++
			return m, pingAllServersCmd(m.store.Servers, m.pingGen)
		case popup.ServerCloseMsg:
			m.serverPopup = nil
			m.pingGen++
			return m, nil
		case popup.FormResultMsg:
			updated, cmd := m.serverPopup.Update(msg)
			m.serverPopup = updated
			return m, cmd
		}
		if res, ok := msg.(serverPingResultMsg); ok {
			if res.gen == m.pingGen {
				if res.online {
					m.serverStatus[res.addr] = popup.PingOnline
				} else {
					m.serverStatus[res.addr] = popup.PingOffline
				}
			}
			return m, nil
		}
		if tick, ok := msg.(serverPingTickMsg); ok {
			if tick.gen == m.pingGen {
				return m, pingAllServersCmd(m.store.Servers, m.pingGen)
			}
			return m, nil
		}
		if isServerInputMsg(msg) {
			updated, cmd := m.serverPopup.Update(msg)
			m.serverPopup = updated
			return m, cmd
		}
	}

	switch msg := msg.(type) {

	case tea.KeyMsg:
		if m.swallowNextKey {
			switch msg.String() {
			case "enter", " ", "space":
				return m, nil
			case "ctrl+c", "ctrl+q":
				m.swallowNextKey = false
				return m, tea.Sequence(closeWSCmd(m.ws), tea.Quit)
			default:
				m.swallowNextKey = false
			}
		}
		switch msg.String() {
		case "ctrl+c", "ctrl+q":
			return m, tea.Sequence(closeWSCmd(m.ws), tea.Quit)
		}
		var cmd tea.Cmd
		m.current, cmd = m.current.Update(msg)
		return m, cmd

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
		if errors.Is(msg.err, ws.ErrUnauthorized) {
			if m.refreshing {
				slog.Info("WS unauthorized, refresh already in progress")
				return m, nil
			}
			slog.Info("WS unauthorized, refreshing token")
			m.refreshing = true
			return m, refreshTokenCmd(m.apiClient, m.auth.RefreshToken)
		}
		slog.Warn("WS dial failed, will retry", "err", msg.err)
		return m, scheduleReconnect()

	case wsReconnectMsg:
		slog.Info("WS reconnecting...")
		return m, dialWSCmd(m.apiClient.WSURL(), m.auth.AccessToken)

	case wsRefreshedMsg:
		slog.Info("token refreshed, reconnecting WS")
		m.refreshing = false
		m.auth.AccessToken = msg.token
		m.apiClient.SetAuth(msg.token)
		return m, dialWSCmd(m.apiClient.WSURL(), msg.token)

	case wsAuthLostMsg:
		slog.Warn("session lost, returning to login")
		m.refreshing = false
		oldWS := m.ws
		m.ws = nil
		m.auth = auth.AuthState{}
		login := views.NewLogin(m.apiClient)
		login.SetNotice("⚠ Session expired — please log in again", false)
		m.current = login
		return m, tea.Batch(
			closeWSCmd(oldWS),
			m.current.Init(),
		)

	case wsmsg.IncomingMsg:
		var cmd tea.Cmd
		m.current, cmd = m.current.Update(msg)
		return m, tea.Batch(cmd, waitWSCmd(m.ws))

	case wsmsg.DMCreatedMsg:
		var cmd tea.Cmd
		m.current, cmd = m.current.Update(msg)
		return m, tea.Batch(cmd, waitWSCmd(m.ws))

	case wsmsg.DMDeletedMsg:
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
		case "open_servers":
			m.serverPopup = m.newServerPopup()
			m.pingGen++
			return m, tea.Batch(
				m.serverPopup.Init(),
				pingAllServersCmd(m.store.Servers, m.pingGen),
			)
		}
		var cmd tea.Cmd
		m.current, cmd = m.current.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	m.current, cmd = m.current.Update(msg)
	return m, cmd
}

func (m Model) newServerPopup() screen.Screen {
	return popup.NewServer(m.store.Servers, m.store.Selected, m.serverStatus)
}

func isServerInputMsg(msg tea.Msg) bool {
	switch msg.(type) {
	case tea.KeyMsg, tea.MouseMsg:
		return true
	}
	return false
}
