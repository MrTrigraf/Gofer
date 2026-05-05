package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gofer/pkg/clipboard"
	"github.com/gofer/tui/auth"
	"github.com/gofer/tui/screen"
	"github.com/gofer/tui/views"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case auth.AuthenticatedMsg:
		m.auth = msg.State
		m.apiClient.SetAuth(msg.State.AccessToken) // ← ДОБАВИТЬ
		m.current = views.NewHome(m.apiClient, m.auth)
		return m, m.current.Init()

	// === NETLINK PINGING ===

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
		// не наш target — отдадим текущему экрану
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
		// также пробросить дальше — у HomeModel свой ClearCopiedMsg для своих target'ов
		var cmd tea.Cmd
		m.current, cmd = m.current.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "ctrl+q":
			return m, tea.Quit
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
			return m, tea.Quit
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
