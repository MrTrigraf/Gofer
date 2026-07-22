package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gofer/pkg/clipboard"
	"github.com/gofer/tui/api"
	"github.com/gofer/tui/screen"
	"github.com/gofer/tui/views/popup"
	"github.com/gofer/tui/wsmsg"
)

// === UPDATE ===

func (m *HomeModel) Update(msg tea.Msg) (screen.Screen, tea.Cmd) {
	// Попап модален ТОЛЬКО для пользовательского ввода.
	if m.popup != nil {
		if res, ok := msg.(popup.ResultMsg); ok {
			m.popup = nil
			return m, m.handlePopupResult(res)
		}
		if res, ok := msg.(popup.FormResultMsg); ok {
			m.popup = nil
			return m, m.handleFormResult(res)
		}
		if isInputMsg(msg) {
			updated, cmd := m.popup.Update(msg)
			m.popup = updated
			return m, cmd
		}
		// не ввод — падаем в основной switch ниже
	}

	switch msg := msg.(type) {

	// === ЗАГРУЗКА ===

	case ChannelsLoadedMsg:
		m.loading = false
		m.channels = msg.Channels
		m.loadErr = nil
		m.actionErr = nil
		if m.cursor >= len(m.channels) {
			m.cursor = 0
		}
		// Если выбранного канала больше нет в списке — сбрасываем выделение.
		if !m.channelExists(m.selectedID) {
			m.selectedID = ""
		}
		return m, m.syncChatTarget()
	case ChannelsLoadErrorMsg:
		m.loading = false
		m.loadErr = msg.Err
		return m, nil

	case DMsLoadedMsg:
		m.dmsLoading = false
		m.dms = msg.DMs
		m.dmsLoadErr = nil
		if m.tab == tabDirect && m.cursor >= len(m.dms) {
			m.cursor = 0
		}
		return m, m.syncChatTarget()
	case DMsLoadErrorMsg:
		m.dmsLoading = false
		m.dmsLoadErr = msg.Err
		return m, nil

	// === ДЕЙСТВИЯ НАД КАНАЛАМИ ===

	case CreateChannelDoneMsg:
		m.actionErr = nil
		m.loading = true
		return m, loadChannelsCmd(m.apiClient)
	case CreateChannelErrorMsg:
		m.actionErr = msg.Err
		m.popup = popup.NewWarning(
			"create_failed:-",
			"Cannot create channel",
			humanizeChannelError(msg.Err),
		)
		return m, nil

	case JoinChannelDoneMsg:
		m.actionErr = nil
		m.loading = true
		return m, loadChannelsCmd(m.apiClient)
	case JoinChannelErrorMsg:
		m.actionErr = msg.Err
		m.popup = popup.NewWarning(
			"join_failed:-",
			"Cannot join channel",
			humanizeChannelError(msg.Err),
		)
		return m, nil

	case LeaveDoneMsg:
		m.actionErr = nil
		m.loading = true
		return m, loadChannelsCmd(m.apiClient)
	case LeaveErrorMsg:
		m.actionErr = msg.Err
		return m, nil

	case DeleteDoneMsg:
		m.actionErr = nil
		m.loading = true
		return m, loadChannelsCmd(m.apiClient)
	case DeleteErrorMsg:
		m.actionErr = msg.Err
		return m, nil

	// === ДЕЙСТВИЯ НАД DM ===

	case StartDMDoneMsg:
		m.actionErr = nil
		m.dmsLoading = true
		return m, loadDMsCmd(m.apiClient)

	case StartDMErrorMsg:
		m.actionErr = msg.Err
		m.popup = popup.NewWarning(
			"start_dm_failed:-",
			"Cannot start direct chat",
			humanizeDMError(msg.Err),
		)
		return m, nil

	case DeleteDMDoneMsg:
		m.actionErr = nil
		m.dmsLoading = true
		// Если выбранный DM удалили — сбросим выделение, иначе UUID-карточка сломается.
		if msg.DMID == m.selectedDMID {
			m.selectedDMID = ""
		}
		return m, tea.Batch(loadDMsCmd(m.apiClient), m.syncChatTarget())
	case DeleteDMErrorMsg:
		m.actionErr = msg.Err
		return m, nil

	// === КЛИПБОРД ===

	case clipboard.CopiedMsg:
		m.copiedTarget = msg.Target
		return m, clipboard.ClearAfterTimeout(msg.Target)
	case clipboard.CopyFailedMsg:
		m.copiedTarget = "fail:" + msg.Target
		return m, clipboard.ClearAfterTimeout(msg.Target)
	case clipboard.ClearCopiedMsg:
		if m.copiedTarget == msg.Target || m.copiedTarget == "fail:"+msg.Target {
			m.copiedTarget = ""
		}
		return m, nil
	case clipboard.UnavailableMsg:
		m.popup = popup.NewWarning(
			"clipboard_unavailable",
			"Clipboard unavailable",
			"Copying is disabled on this system.\n\nInstall xclip, xsel or wl-copy to enable it.",
		)
		return m, nil

	// === ВВОД ===

	case tea.KeyMsg:
		if m.chat.HasTarget() {
			switch msg.String() {
			case "up", "down", "tab", "esc", "enter":
				// падаем дальше в навигационный switch
			default:
				var cmd tea.Cmd
				m.chat, cmd = m.chat.Update(msg)
				return m, cmd
			}
		}

		switch msg.String() {
		case "a":
			if !m.addMode {
				m.addMode = true
			}
			return m, nil
		case "l":
			if m.addMode && m.tab == tabChannels {
				if m.selectedID == "" {
					return m, nil
				}
				m.openLeaveChannelConfirm(m.selectedID)
			}
			return m, nil
		case "d":
			if !m.addMode {
				return m, nil
			}
			if m.tab == tabChannels && m.selectedID != "" {
				m.openDeleteChannelConfirm(m.selectedID)
			}
			if m.tab == tabDirect && m.selectedDMID != "" {
				m.openDeleteDMConfirm(m.selectedDMID)
			}
			return m, nil
		case "esc":
			if m.addMode {
				m.addMode = false
				return m, nil
			}
			if m.chat.HasTarget() {
				m.chat.Clear()
				m.selectedID = ""
				m.selectedDMID = ""
				return m, nil
			}
			return m, nil
		case "tab":
			m.toggleTab()
			return m, nil
		case "up":
			var cmd tea.Cmd
			if m.cursor > 0 {
				m.cursor--
				cmd = m.syncSelectionFromCursor()
				m.ensureCursorVisible()
			}
			return m, cmd
		case "down":
			var cmd tea.Cmd
			if m.cursor < m.listLen()-1 {
				m.cursor++
				cmd = m.syncSelectionFromCursor()
				m.ensureCursorVisible()
			}
			return m, cmd
		case "enter":
			if m.chat.HasTarget() {
				var cmd tea.Cmd
				m.chat, cmd = m.chat.Update(msg)
				return m, cmd
			}
			return m, m.activateCursor()
		}

	case tea.MouseMsg:
		if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
			if m.addMode {
				var cmd tea.Cmd
				m.addListVP, cmd = m.addListVP.Update(msg)
				return m, cmd
			}
			if m.chat.HasTarget() {
				var cmd tea.Cmd
				m.chat, cmd = m.chat.Update(msg)
				return m, cmd
			}
			return m, nil
		}
		if msg.Action != tea.MouseActionPress {
			return m, nil
		}
		id := screen.HitTest(m.hitboxes, msg.X, msg.Y)
		switch {
		case id == "tab_direct":
			m.tab = tabDirect
			m.cursor = 0

		case id == "tab_channels":
			m.tab = tabChannels
			m.cursor = 0

		case id == "sidebar_add":
			m.addMode = true

		case id == "sidebar_back":
			m.addMode = false

		case id == "add_channel_create":
			m.popup = popup.NewForm(
				"create_channel:-",
				"Create channel",
				"Channel name:",
				"e.g. general",
				"Create",
				32,
			)
			return m, m.popup.Init()

		case id == "add_channel_join":
			m.popup = popup.NewForm(
				"join_channel:-",
				"Join channel",
				"Channel ID:",
				"e.g. 6679e14e-e7a5-42eb-9198-3dd7a34f3013",
				"Join",
				40,
			)
			return m, m.popup.Init()

		case id == "add_channel_delete":
			if m.selectedID == "" {
				m.popup = popup.NewWarning(
					"no_channel_selected:-",
					"No channel selected",
					"Select a channel first, then click delete.",
				)
			} else {
				m.openDeleteChannelConfirm(m.selectedID)
			}

		case id == "add_dm_start":
			m.popup = popup.NewForm(
				"start_dm:-",
				"Start direct chat",
				"User ID:",
				"e.g. f8d82de2-242b-4df1-a440-63db4d62f661",
				"Start",
				40,
			)
			return m, m.popup.Init()

		case id == "my_uuid":
			return m, clipboard.CopyCmd("my_uuid", m.state.UserID)

		case id == "channel_uuid":
			if m.selectedID != "" {
				return m, clipboard.CopyCmd("channel_uuid", m.selectedID)
			}

		case id == "dm_user_uuid":
			if dm := m.findDM(m.selectedDMID); dm != nil {
				return m, clipboard.CopyCmd("dm_user_uuid", dm.OtherUserID)
			}

		case strings.HasPrefix(id, "channel_leave_"):
			channelID := strings.TrimPrefix(id, "channel_leave_")
			m.openLeaveChannelConfirm(channelID)

		case strings.HasPrefix(id, "dm_delete_"):
			dmID := strings.TrimPrefix(id, "dm_delete_")
			m.openDeleteDMConfirm(dmID)

		case strings.HasPrefix(id, "channel_"):
			channelID := strings.TrimPrefix(id, "channel_")
			m.selectedID = channelID
			for i, ch := range m.channels {
				if ch.ID == channelID {
					m.cursor = i
					break
				}
			}
			return m, m.syncChatTarget()

		case strings.HasPrefix(id, "dm_"):
			dmID := strings.TrimPrefix(id, "dm_")
			m.selectedDMID = dmID
			for i, dm := range m.dms {
				if dm.ID == dmID {
					m.cursor = i
					break
				}
			}
			return m, m.syncChatTarget()
		}

	// === ЧАТ И WS ===

	case wsmsg.DMCreatedMsg:
		m.dmsLoading = true
		return m, loadDMsCmd(m.apiClient)
	}

	if isChatMsg(msg) {
		var cmd tea.Cmd
		m.chat, cmd = m.chat.Update(msg)
		return m, cmd
	}
	return m, nil
}

// === РЕЗУЛЬТАТЫ ПОПАПОВ ===

func (m *HomeModel) handlePopupResult(res popup.ResultMsg) tea.Cmd {
	parts := strings.SplitN(res.Action, ":", 2)
	if len(parts) != 2 {
		return nil
	}
	kind, targetID := parts[0], parts[1]

	if !res.Confirmed {
		return nil // юзер отменил — ничего не делаем
	}

	switch kind {
	case "leave_channel":
		return leaveChannelCmd(m.apiClient, targetID)
	case "delete_channel":
		return deleteChannelCmd(m.apiClient, targetID)
	case "delete_dm":
		return deleteDMCmd(m.apiClient, targetID)
	}
	return nil
}

func (m *HomeModel) handleFormResult(res popup.FormResultMsg) tea.Cmd {
	if !res.Confirmed {
		return nil
	}
	parts := strings.SplitN(res.Action, ":", 2)
	if len(parts) != 2 {
		return nil
	}
	kind := parts[0]

	switch kind {
	case "create_channel":
		name := strings.TrimSpace(res.Value)
		if name == "" {
			m.popup = popup.NewWarning(
				"name_empty:-",
				"Cannot create channel",
				"Name cannot be empty.",
			)
			return nil
		}
		return createChannelCmd(m.apiClient, name)

	case "join_channel":
		channelID := strings.TrimSpace(res.Value)
		if channelID == "" {
			m.popup = popup.NewWarning(
				"id_empty:-",
				"Cannot join channel",
				"Channel ID cannot be empty.",
			)
			return nil
		}
		return joinChannelCmd(m.apiClient, channelID)

	case "start_dm":
		userID := strings.TrimSpace(res.Value)
		if userID == "" {
			m.popup = popup.NewWarning(
				"id_empty:-",
				"Cannot start direct chat",
				"User ID cannot be empty.",
			)
			return nil
		}
		return startDMCmd(m.apiClient, userID)
	}
	return nil
}

// === ОТКРЫВАТЕЛИ ПОДТВЕРЖДЕНИЙ ===

// openLeaveChannelConfirm — открывает попап подтверждения выхода из канала.
func (m *HomeModel) openLeaveChannelConfirm(channelID string) {
	name := m.channelName(channelID)
	if name == "" {
		return // канала нет в текущем списке — игнорируем
	}
	m.popup = popup.NewConfirm(
		"leave_channel:"+channelID,
		"Leave channel",
		fmt.Sprintf("Leave channel #%s?", name),
		"Leave",
		false,
	)
}

// openDeleteChannelConfirm — либо confirm (создатель), либо warning (не создатель).
func (m *HomeModel) openDeleteChannelConfirm(channelID string) {
	var ch api.Channel
	found := false
	for _, c := range m.channels {
		if c.ID == channelID {
			ch = c
			found = true
			break
		}
	}
	if !found {
		return
	}

	if ch.CreatedBy != m.state.UserID {
		m.popup = popup.NewWarning(
			"not_creator:"+channelID,
			"Cannot delete",
			"You are not the channel creator.",
		)
		return
	}

	m.popup = popup.NewConfirm(
		"delete_channel:"+channelID,
		"Delete channel",
		fmt.Sprintf("Delete channel #%s? This cannot be undone.", ch.Name),
		"Delete",
		true,
	)
}

func (m *HomeModel) openDeleteDMConfirm(dmID string) {
	var username string
	for _, dm := range m.dms {
		if dm.ID == dmID {
			username = dm.OtherUsername
			break
		}
	}
	if username == "" {
		return // DM нет в списке — игнор
	}
	m.popup = popup.NewConfirm(
		"delete_dm:"+dmID,
		"Delete DM",
		fmt.Sprintf("Delete DM with %s? This cannot be undone.", username),
		"Delete",
		true,
	)
}

func isInputMsg(msg tea.Msg) bool {
	switch msg.(type) {
	case tea.KeyMsg, tea.MouseMsg:
		return true
	}
	return false
}
