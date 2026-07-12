package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/gofer/tui/screen"
	"github.com/gofer/tui/styles"
)

// addListLabelWidth — ширина колонки с именем в add-режиме, до которой
// добивается padding перед action-кнопкой ([←] exit / [×] del).
// FIX(TODO#4): была магическая 40 в двух местах.
const addListLabelWidth = 40

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

// === САЙДБАР ===

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

	m.hitboxes = append(
		m.hitboxes,
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
			{label: "[×] delete", id: "add_channel_delete"},
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
		label = styles.StyleAccent.Render("[×] BACK")
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

// === ОБЛАСТЬ ЧАТА ===

func (m *HomeModel) renderChatArea() string {
	chatW := m.width - styles.SidebarWidth - 1
	chatH := m.height

	if m.addMode {
		return m.renderAddChatArea(chatW, chatH)
	}

	if m.chat.HasTarget() {
		return m.chat.View(chatW, chatH)
	}

	return lipgloss.Place(
		chatW, chatH,
		lipgloss.Center, lipgloss.Center,
		styles.StyleFaint.Render("Select a channel or chat"),
	)
}

// === ADD-РЕЖИМ ===

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

		padWidth := addListLabelWidth - lipgloss.Width(name)
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

		action := styles.StyleAccent.Render("[×] del")

		padWidth := addListLabelWidth - lipgloss.Width(label)
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

// === UUID-КАРТОЧКИ ===

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
