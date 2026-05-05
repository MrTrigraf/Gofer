package views

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gofer/pkg/clipboard"
	"github.com/gofer/tui/api"
	"github.com/gofer/tui/auth"
	"github.com/gofer/tui/screen"
	"github.com/gofer/tui/styles"
	"github.com/gofer/tui/views/popup"
)

type activeTab int

// Порядок: DIRECT первый (слева), CHANNELS второй (справа).
const (
	tabDirect activeTab = iota
	tabChannels
)

const chatAreaListReserve = 13
const chatAreaListOffset = 3

// === СООБЩЕНИЯ ===

type ChannelsLoadedMsg struct {
	Channels []api.Channel
}

type ChannelsLoadErrorMsg struct {
	Err error
}

type DMsLoadedMsg struct {
	DMs []api.DirectChat
}

type DMsLoadErrorMsg struct {
	Err error
}

// Ответы от action-команд.
type LeaveDoneMsg struct {
	ChannelID string
}

type CreateChannelDoneMsg struct {
	Channel api.Channel
}

type CreateChannelErrorMsg struct {
	Err error
}

type JoinChannelDoneMsg struct {
	ChannelID string
}

type JoinChannelErrorMsg struct {
	Err error
}

type StartDMDoneMsg struct {
	DM api.DirectChat
}

type StartDMErrorMsg struct {
	Err error
}

type LeaveErrorMsg struct {
	Err error
}

type DeleteDoneMsg struct {
	ChannelID string
}

type DeleteErrorMsg struct {
	Err error
}

type DeleteDMDoneMsg struct {
	DMID string
}

type DeleteDMErrorMsg struct {
	Err error
}

// === МОДЕЛЬ ===

type HomeModel struct {
	apiClient *api.Client
	state     auth.AuthState

	tab activeTab

	// Channels
	channels   []api.Channel
	loadErr    error
	loading    bool
	selectedID string

	// DMs
	dms          []api.DirectChat
	dmsLoadErr   error
	dmsLoading   bool
	selectedDMID string

	// cursor — индекс элемента ПОД КУРСОРОМ в активном списке.
	cursor int

	// addMode — true, когда юзер вошёл в режим управления.
	addMode bool

	// popup — активный модальный попап. nil = нет попапа.
	popup     screen.Screen
	addListVP viewport.Model
	// actionErr — последняя ошибка действия (leave/delete). Сбрасывается при успехе.
	actionErr    error
	copiedTarget string

	width, height    int
	originX, originY int

	hitboxes []screen.Hitbox
}

func NewHome(apiClient *api.Client, state auth.AuthState) *HomeModel {
	return &HomeModel{
		apiClient: apiClient,
		state:     state,
		tab:       tabDirect,
		addListVP: viewport.New(0, 0),
	}
}

// === screen.Screen ===

func (m *HomeModel) Init() tea.Cmd {
	m.loading = true
	m.dmsLoading = true
	return tea.Batch(
		loadChannelsCmd(m.apiClient),
		loadDMsCmd(m.apiClient),
	)
}

func (m *HomeModel) SetSize(width, height int) { m.width, m.height = width, height }
func (m *HomeModel) SetOrigin(x, y int)        { m.originX, m.originY = x, y }

func (m *HomeModel) Hitboxes() []screen.Hitbox {
	if m.popup != nil {
		return append(m.hitboxes, m.popup.Hitboxes()...)
	}
	return m.hitboxes
}

// === КОМАНДЫ ===

func loadChannelsCmd(client *api.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		channels, err := client.ListChannels(ctx)
		if err != nil {
			return ChannelsLoadErrorMsg{Err: err}
		}
		return ChannelsLoadedMsg{Channels: channels}
	}
}

func loadDMsCmd(client *api.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		dms, err := client.ListDMs(ctx)
		if err != nil {
			return DMsLoadErrorMsg{Err: err}
		}
		return DMsLoadedMsg{DMs: dms}
	}
}

func leaveChannelCmd(client *api.Client, channelID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := client.LeaveChannel(ctx, channelID); err != nil {
			return LeaveErrorMsg{Err: err}
		}
		return LeaveDoneMsg{ChannelID: channelID}
	}
}

func createChannelCmd(client *api.Client, name string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		ch, err := client.CreateChannel(ctx, name)
		if err != nil {
			return CreateChannelErrorMsg{Err: err}
		}
		return CreateChannelDoneMsg{Channel: ch}
	}
}

func joinChannelCmd(client *api.Client, channelID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := client.JoinChannel(ctx, channelID); err != nil {
			return JoinChannelErrorMsg{Err: err}
		}
		return JoinChannelDoneMsg{ChannelID: channelID}
	}
}

func deleteChannelCmd(client *api.Client, channelID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := client.DeleteChannel(ctx, channelID); err != nil {
			return DeleteErrorMsg{Err: err}
		}
		return DeleteDoneMsg{ChannelID: channelID}
	}
}

func deleteDMCmd(client *api.Client, dmID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := client.DeleteDM(ctx, dmID); err != nil {
			return DeleteDMErrorMsg{Err: err}
		}
		return DeleteDMDoneMsg{DMID: dmID}
	}
}

func startDMCmd(client *api.Client, userID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		dm, err := client.StartDM(ctx, userID)
		if err != nil {
			return StartDMErrorMsg{Err: err}
		}
		return StartDMDoneMsg{DM: dm}
	}
}

// === UPDATE ===

func (m *HomeModel) Update(msg tea.Msg) (screen.Screen, tea.Cmd) {
	// Если активен попап — он ест все события, пока жив.
	if m.popup != nil {
		// Confirm-результат
		if res, ok := msg.(popup.ResultMsg); ok {
			m.popup = nil
			return m, m.handlePopupResult(res)
		}
		// Form-результат
		if res, ok := msg.(popup.FormResultMsg); ok {
			m.popup = nil
			return m, m.handleFormResult(res)
		}
		updated, cmd := m.popup.Update(msg)
		m.popup = updated
		return m, cmd
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
		return m, nil
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
		return m, nil
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
		return m, loadDMsCmd(m.apiClient)
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

	// === ВВОД ===

	case tea.KeyMsg:
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
			}
			return m, nil
		case "tab":
			m.toggleTab()
			return m, nil
		case "up":
			if m.cursor > 0 {
				m.cursor--
				m.syncSelectionFromCursor()
				m.ensureCursorVisible()
			}
			return m, nil
		case "down":
			if m.cursor < m.listLen()-1 {
				m.cursor++
				m.syncSelectionFromCursor()
				m.ensureCursorVisible()
			}
			return m, nil
		case "enter":
			m.activateCursor()
			return m, nil
		}

	case tea.MouseMsg:
		// Wheel-события перенаправляем во viewport списка (только в add-режиме).
		if m.addMode && (msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown) {
			var cmd tea.Cmd
			m.addListVP, cmd = m.addListVP.Update(msg)
			return m, cmd
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
		case strings.HasPrefix(id, "dm_"):
			dmID := strings.TrimPrefix(id, "dm_")
			m.selectedDMID = dmID
			for i, dm := range m.dms {
				if dm.ID == dmID {
					m.cursor = i
					break
				}
			}
		}
	}
	return m, nil
}

// handlePopupResult — разбирает тег Action и запускает соответствующую команду.
// Формат Action: "kind:targetID" (например, "leave_channel:abc-123").
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

// channelName — имя канала по ID или пустая строка.
func (m *HomeModel) channelName(id string) string {
	for _, ch := range m.channels {
		if ch.ID == id {
			return ch.Name
		}
	}
	return ""
}

// findDM — DM по ID или nil.
func (m *HomeModel) findDM(id string) *api.DirectChat {
	if id == "" {
		return nil
	}
	for i := range m.dms {
		if m.dms[i].ID == id {
			return &m.dms[i]
		}
	}
	return nil
}

// channelExists — есть ли канал с таким ID в текущем списке.
func (m *HomeModel) channelExists(id string) bool {
	if id == "" {
		return false
	}
	return m.channelName(id) != ""
}

func (m *HomeModel) toggleTab() {
	if m.tab == tabChannels {
		m.tab = tabDirect
	} else {
		m.tab = tabChannels
	}
	m.cursor = 0
}

func (m *HomeModel) listLen() int {
	if m.tab == tabChannels {
		return len(m.channels)
	}
	return len(m.dms)
}

func (m *HomeModel) activateCursor() {
	if m.tab == tabChannels {
		if m.cursor >= 0 && m.cursor < len(m.channels) {
			m.selectedID = m.channels[m.cursor].ID
		}
		return
	}
	if m.cursor >= 0 && m.cursor < len(m.dms) {
		m.selectedDMID = m.dms[m.cursor].ID
	}
}

func (m *HomeModel) syncSelectionFromCursor() {
	if !m.addMode {
		return
	}
	m.activateCursor()
}

func (m *HomeModel) ensureCursorVisible() {
	if !m.addMode {
		return
	}
	h := m.addListVP.Height
	if h <= 0 {
		return
	}
	if m.cursor < m.addListVP.YOffset {
		m.addListVP.YOffset = m.cursor
	} else if m.cursor > m.addListVP.YOffset+h-1 {
		m.addListVP.YOffset = m.cursor - h + 1
	}
}

// === VIEW ===

func (m *HomeModel) View() string {
	m.hitboxes = m.hitboxes[:0]

	sidebar := m.renderSidebar()
	chat := m.renderChatArea()
	home := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, chat)

	if m.popup != nil {
		m.popup.SetSize(m.width, m.height)
		m.popup.SetOrigin(m.originX, m.originY)
		return m.popup.View()
	}

	return home
}

func (m *HomeModel) renderSidebar() string {
	sbWidth := styles.SidebarWidth
	sbHeight := m.height

	tabs := m.renderTabs(sbWidth)
	divider := styles.StyleDivider.Render(strings.Repeat("─", sbWidth-1))
	list := m.renderList(sbWidth)
	addBtn := m.renderAddButton(sbWidth)

	tabsH := lipgloss.Height(tabs)
	dividerH := 1
	addH := lipgloss.Height(addBtn)

	availableForList := sbHeight - tabsH - dividerH - addH
	if availableForList < 1 {
		availableForList = 1
	}
	listStretched := lipgloss.NewStyle().
		Height(availableForList).
		Render(list)

	inner := lipgloss.JoinVertical(
		lipgloss.Left,
		tabs,
		divider,
		listStretched,
		addBtn,
	)

	return styles.StyleSidebar.
		Width(sbWidth - 1).
		Height(sbHeight).
		Render(inner)
}

func (m *HomeModel) renderTabs(sbWidth int) string {
	var dmTab, chTab string
	if m.tab == tabDirect {
		dmTab = styles.StyleTabActive.Render("[DIRECT]")
		chTab = styles.StyleTabInactive.Render(" CHANNELS ")
	} else {
		dmTab = styles.StyleTabInactive.Render(" DIRECT ")
		chTab = styles.StyleTabActive.Render("[CHANNELS]")
	}
	sep := styles.StyleDivider.Render("│")
	row := " " + dmTab + sep + chTab

	dmStart := m.originX + 1
	dmEnd := dmStart + lipgloss.Width(dmTab) - 1
	chStart := dmEnd + 1 + lipgloss.Width(sep)
	chEnd := chStart + lipgloss.Width(chTab) - 1

	m.hitboxes = append(m.hitboxes,
		screen.Hitbox{X1: dmStart, Y1: m.originY, X2: dmEnd, Y2: m.originY, ID: "tab_direct"},
		screen.Hitbox{X1: chStart, Y1: m.originY, X2: chEnd, Y2: m.originY, ID: "tab_channels"},
	)

	return row
}

func (m *HomeModel) renderList(sbWidth int) string {
	if m.addMode {
		return m.renderAddActions(sbWidth)
	}
	if m.tab == tabChannels {
		return m.renderChannelsList(sbWidth)
	}
	return m.renderDMsList(sbWidth)
}

func (m *HomeModel) renderAddActions(sbWidth int) string {
	type action struct {
		label string
		id    string
	}

	var actions []action
	if m.tab == tabChannels {
		actions = []action{
			{label: "[+] create", id: "add_channel_create"},
			{label: "[→] join", id: "add_channel_join"},
			{label: "[🗑] delete", id: "add_channel_delete"},
		}
	} else {
		actions = []action{
			{label: "[+] start DM", id: "add_dm_start"},
		}
	}

	var lines []string
	for i, a := range actions {
		lineY := m.originY + 2 + i
		rendered := "  " + styles.StyleItemInactive.Render(a.label)
		lines = append(lines, rendered)

		m.hitboxes = append(m.hitboxes, screen.Hitbox{
			X1: m.originX, Y1: lineY,
			X2: m.originX + sbWidth - 2, Y2: lineY,
			ID: a.id,
		})
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m *HomeModel) renderChannelsList(sbWidth int) string {
	switch {
	case m.loading:
		return styles.StyleFaint.Render("  loading...")
	case m.loadErr != nil:
		return styles.StyleDanger.Render("  ⚠ Failed to load")
	case len(m.channels) == 0:
		return styles.StyleFaint.Render("  (no channels yet)")
	}

	var lines []string
	for i, ch := range m.channels {
		lineY := m.originY + 2 + i

		var prefix string
		if i == m.cursor {
			prefix = styles.StyleAccent.Render("▸ ")
		} else {
			prefix = "  "
		}

		label := "# " + ch.Name
		if ch.ID == m.selectedID {
			label = styles.StyleItemActive.Render(label)
		} else {
			label = styles.StyleItemInactive.Render(label)
		}

		lines = append(lines, prefix+label)

		m.hitboxes = append(m.hitboxes, screen.Hitbox{
			X1: m.originX, Y1: lineY,
			X2: m.originX + sbWidth - 2, Y2: lineY,
			ID: "channel_" + ch.ID,
		})
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m *HomeModel) renderDMsList(sbWidth int) string {
	switch {
	case m.dmsLoading:
		return styles.StyleFaint.Render("  loading...")
	case m.dmsLoadErr != nil:
		return styles.StyleDanger.Render("  ⚠ Failed to load")
	case len(m.dms) == 0:
		return styles.StyleFaint.Render("  (no direct chats)")
	}

	var lines []string
	for i, dm := range m.dms {
		lineY := m.originY + 2 + i

		var prefix string
		if i == m.cursor {
			prefix = styles.StyleAccent.Render("▸ ")
		} else {
			prefix = "  "
		}

		icon := styles.StyleOffline.Render("○ ")
		name := dm.OtherUsername
		if dm.ID == m.selectedDMID {
			name = styles.StyleItemActive.Render(name)
		} else {
			name = styles.StyleItemInactive.Render(name)
		}

		lines = append(lines, prefix+icon+name)

		m.hitboxes = append(m.hitboxes, screen.Hitbox{
			X1: m.originX, Y1: lineY,
			X2: m.originX + sbWidth - 2, Y2: lineY,
			ID: "dm_" + dm.ID,
		})
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m *HomeModel) renderAddButton(sbWidth int) string {
	var label, hitboxID string
	if m.addMode {
		label = styles.StyleAccent.Render("[✕] BACK")
		hitboxID = "sidebar_back"
	} else {
		label = styles.StyleAccent.Render("[+] ADD")
		hitboxID = "sidebar_add"
	}
	labelW := lipgloss.Width(label)

	addY := m.originY + m.height - 1
	addX2 := m.originX + sbWidth - 2
	addX1 := addX2 - labelW + 1
	m.hitboxes = append(m.hitboxes, screen.Hitbox{
		X1: addX1, Y1: addY,
		X2: addX2, Y2: addY,
		ID: hitboxID,
	})

	pad := sbWidth - 2 - labelW
	if pad < 0 {
		pad = 0
	}
	return strings.Repeat(" ", pad) + label
}

func (m *HomeModel) renderChatArea() string {
	chatW := m.width - styles.SidebarWidth
	chatH := m.height

	if m.addMode {
		return m.renderAddChatArea(chatW, chatH)
	}

	var content string
	switch {
	case m.tab == tabChannels && m.selectedID != "":
		content = m.renderChannelChatHeader()
	case m.tab == tabDirect && m.selectedDMID != "":
		content = m.renderDMChatHeader()
	default:
		content = styles.StyleFaint.Render("Select a channel or chat")
	}

	return lipgloss.Place(
		chatW, chatH,
		lipgloss.Center, lipgloss.Center,
		content,
	)
}

func (m *HomeModel) renderAddChatArea(chatW, chatH int) string {
	chatOriginX := m.originX + styles.SidebarWidth
	chatOriginY := m.originY

	var header, list, uuidCard, uuidHitboxID, uuidValue string

	if m.tab == tabChannels {
		header = styles.StyleTitle.Render("Your channels:")
		list = m.renderAddChannelsList(chatOriginX, chatOriginY+chatAreaListOffset)
		uuidCard = m.renderChannelUUIDCard()
		uuidHitboxID = "channel_uuid"
		uuidValue = m.selectedID
	} else {
		header = styles.StyleTitle.Render("Your chats:")
		list = m.renderAddDMsList(chatOriginX, chatOriginY+chatAreaListOffset)
		uuidCard = m.renderDMUUIDCard()
		uuidHitboxID = "dm_user_uuid"
		if dm := m.findDM(m.selectedDMID); dm != nil {
			uuidValue = dm.OtherUserID
		}
	}

	divider := styles.StyleDivider.Render(strings.Repeat("─", chatW-4))
	myUUIDCard := m.renderMyUUIDCard()

	listH := chatH - chatAreaListReserve
	if listH < 1 {
		listH = 1
	}

	m.addListVP.Width = chatW - 4
	m.addListVP.Height = listH
	m.addListVP.SetContent(list)

	inner := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		m.addListVP.View(),
		"",
		divider,
		uuidCard,
		"",
		divider,
		myUUIDCard,
	)

	rendered := lipgloss.NewStyle().
		Width(chatW).
		Height(chatH).
		Padding(1, 2).
		Render(inner)

	// Базовая X-координата текста внутри chat-area (с учётом padding left=2).
	textStartX := chatOriginX + 2

	// === Hitbox карточки выбранного ресурса (channel/dm) ===
	// UUID-строка идёт второй строкой uuidCard (после заголовка).
	// Y до неё считаем динамически.
	if uuidValue != "" {
		uuidLineY := chatOriginY + 1 + // padding top
			lipgloss.Height(header) +
			1 + // blank
			listH +
			1 + // blank
			1 + // divider
			1 // header inside uuidCard ("# name UUID:")
		uuidEndX := textStartX + lipgloss.Width(uuidValue) - 1
		m.hitboxes = append(m.hitboxes, screen.Hitbox{
			X1: textStartX, Y1: uuidLineY,
			X2: uuidEndX, Y2: uuidLineY,
			ID: uuidHitboxID,
		})
	}

	// === Hitbox карточки "Your UUID" ===
	myUUIDLineY := chatOriginY + 1 + // padding top
		lipgloss.Height(header) +
		1 + // blank
		listH +
		1 + // blank
		1 + // divider
		lipgloss.Height(uuidCard) +
		1 + // blank
		1 + // divider
		1 // "Your UUID:" header inside myUUIDCard

	myUUIDEndX := textStartX + lipgloss.Width(m.state.UserID) - 1
	m.hitboxes = append(m.hitboxes, screen.Hitbox{
		X1: textStartX, Y1: myUUIDLineY,
		X2: myUUIDEndX, Y2: myUUIDLineY,
		ID: "my_uuid",
	})

	return rendered
}

func (m *HomeModel) renderAddChannelsList(originX, originY int) string {
	if len(m.channels) == 0 {
		return styles.StyleFaint.Render("(no channels yet)")
	}

	yOffset := m.addListVP.YOffset
	vpTop := originY
	vpBottom := originY + m.addListVP.Height - 1

	var lines []string
	for i, ch := range m.channels {
		name := "# " + ch.Name
		if ch.ID == m.selectedID {
			name = styles.StyleItemActive.Render(name)
		} else {
			name = styles.StyleItemInactive.Render(name)
		}

		action := styles.StyleAccent.Render("[←] exit")

		padWidth := 40 - lipgloss.Width(name)
		if padWidth < 1 {
			padWidth = 1
		}
		line := name + strings.Repeat(" ", padWidth) + action
		lines = append(lines, line)

		// Физический Y строки на экране с учётом скролла.
		physY := originY + i - yOffset
		// Регистрируем hitbox только если строка в видимой зоне viewport'а.
		if physY < vpTop || physY > vpBottom {
			continue
		}

		nameEndX := originX + lipgloss.Width(name) - 1
		m.hitboxes = append(m.hitboxes, screen.Hitbox{
			X1: originX, Y1: physY,
			X2: nameEndX, Y2: physY,
			ID: "channel_" + ch.ID,
		})

		actionX1 := originX + lipgloss.Width(name) + padWidth
		actionX2 := actionX1 + lipgloss.Width(action) - 1
		m.hitboxes = append(m.hitboxes, screen.Hitbox{
			X1: actionX1, Y1: physY,
			X2: actionX2, Y2: physY,
			ID: "channel_leave_" + ch.ID,
		})
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m *HomeModel) renderAddDMsList(originX, originY int) string {
	if len(m.dms) == 0 {
		return styles.StyleFaint.Render("(no direct chats)")
	}

	yOffset := m.addListVP.YOffset
	vpTop := originY
	vpBottom := originY + m.addListVP.Height - 1

	var lines []string
	for i, dm := range m.dms {
		icon := styles.StyleOffline.Render("○ ")
		name := dm.OtherUsername
		if dm.ID == m.selectedDMID {
			name = styles.StyleItemActive.Render(name)
		} else {
			name = styles.StyleItemInactive.Render(name)
		}
		label := icon + name

		action := styles.StyleAccent.Render("[✕] del")

		padWidth := 40 - lipgloss.Width(label)
		if padWidth < 1 {
			padWidth = 1
		}
		line := label + strings.Repeat(" ", padWidth) + action
		lines = append(lines, line)

		physY := originY + i - yOffset
		if physY < vpTop || physY > vpBottom {
			continue
		}

		nameEndX := originX + lipgloss.Width(label) - 1
		m.hitboxes = append(m.hitboxes, screen.Hitbox{
			X1: originX, Y1: physY,
			X2: nameEndX, Y2: physY,
			ID: "dm_" + dm.ID,
		})

		actionX1 := originX + lipgloss.Width(label) + padWidth
		actionX2 := actionX1 + lipgloss.Width(action) - 1
		m.hitboxes = append(m.hitboxes, screen.Hitbox{
			X1: actionX1, Y1: physY,
			X2: actionX2, Y2: physY,
			ID: "dm_delete_" + dm.ID,
		})
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m *HomeModel) renderChannelUUIDCard() string {
	if m.selectedID == "" {
		return styles.StyleFaint.Render("Click channel to see its ID")
	}
	name := m.channelName(m.selectedID)
	header := styles.StyleFaint.Render("# " + name + " ID:")
	uuid := styles.StyleItemActive.Render(m.selectedID)

	var feedback string
	switch m.copiedTarget {
	case "channel_uuid":
		feedback = "  " + styles.StyleOK.Render("✓ copied")
	case "fail:channel_uuid":
		feedback = "  " + styles.StyleDanger.Render("⚠ unavailable")
	default:
		feedback = "  " + styles.StyleFaint.Render("(copy)")
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, uuid+feedback)
}

func (m *HomeModel) renderMyUUIDCard() string {
	header := styles.StyleFaint.Render("Your ID:")

	uuidLine := styles.StyleItemActive.Render(m.state.UserID)

	var feedback string
	switch m.copiedTarget {
	case "my_uuid":
		feedback = "  " + styles.StyleOK.Render("✓ copied")
	case "fail:my_uuid":
		feedback = "  " + styles.StyleDanger.Render("⚠ unavailable")
	default:
		feedback = "  " + styles.StyleFaint.Render("(copy)")
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		uuidLine+feedback,
	)
}

func (m *HomeModel) renderDMUUIDCard() string {
	dm := m.findDM(m.selectedDMID)
	if dm == nil {
		return styles.StyleFaint.Render("Click chat to see user ID")
	}
	header := styles.StyleFaint.Render(dm.OtherUsername + " ID:")
	uuid := styles.StyleItemActive.Render(dm.OtherUserID)

	var feedback string
	switch m.copiedTarget {
	case "dm_user_uuid":
		feedback = "  " + styles.StyleOK.Render("✓ copied")
	case "fail:dm_user_uuid":
		feedback = "  " + styles.StyleDanger.Render("⚠ unavailable")
	default:
		feedback = "  " + styles.StyleFaint.Render("(copy)")
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, uuid+feedback)
}

func (m *HomeModel) renderChannelChatHeader() string {
	name := m.channelName(m.selectedID)
	if name == "" {
		name = "unknown"
	}

	title := styles.StyleTitle.Render(fmt.Sprintf("# %s", name))
	hint := styles.StyleFaint.Render("(chat coming in 8.3 with WebSocket)")
	return lipgloss.JoinVertical(lipgloss.Center, title, "", hint)
}

func (m *HomeModel) renderDMChatHeader() string {
	var username string
	for _, dm := range m.dms {
		if dm.ID == m.selectedDMID {
			username = dm.OtherUsername
			break
		}
	}
	if username == "" {
		username = "unknown"
	}

	title := styles.StyleTitle.Render(fmt.Sprintf("@ %s", username))
	hint := styles.StyleFaint.Render("(chat coming in 8.3 with WebSocket)")
	return lipgloss.JoinVertical(lipgloss.Center, title, "", hint)
}

func humanizeChannelError(err error) string {
	switch {
	case errors.Is(err, api.ErrNotFound):
		return "Channel not found."
	case errors.Is(err, api.ErrForbidden):
		return "You don't have permission for this action."
	case errors.Is(err, api.ErrBadRequest):
		return err.Error()
	case errors.Is(err, api.ErrUnreachable):
		return "Server unreachable."
	case errors.Is(err, api.ErrServer):
		return "Server error, try again later."
	}
	return err.Error()
}

func humanizeDMError(err error) string {
	switch {
	case errors.Is(err, api.ErrNotFound):
		return "User not found."
	case errors.Is(err, api.ErrConflict):
		return "You already have a direct chat with this user."
	case errors.Is(err, api.ErrBadRequest):
		return err.Error()
	case errors.Is(err, api.ErrUnreachable):
		return "Server unreachable."
	case errors.Is(err, api.ErrServer):
		return "Server error, try again later."
	}
	return err.Error()
}
