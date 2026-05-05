package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/gofer/tui/screen"
	"github.com/gofer/tui/styles"
)

// renderTooSmall — экран-заглушка по центру терминала (макет №11).
func (m Model) renderTooSmall() string {
	msg := lipgloss.JoinVertical(
		lipgloss.Center,
		styles.StyleDanger.Render("⚠ Window too small."),
		styles.StyleFaint.Render(fmt.Sprintf("Min size: %d x %d", minWidth, minHeight)),
		styles.StyleFaint.Render(fmt.Sprintf("Current:  %d x %d", m.width, m.height)),
	)
	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		msg,
	)
}

// renderApp — основной каркас: AppFrame + Header + Body + Footer.
func (m Model) renderApp() string {
	*m.hitboxes = (*m.hitboxes)[:0]

	innerWidth := m.width - 2
	innerHeight := m.height - 2

	bodyHeight := innerHeight - 2 - 2
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	header := m.renderHeader(innerWidth)
	body := m.renderBody(innerWidth, bodyHeight)
	footer := m.renderFooter(innerWidth)

	inner := lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
	return styles.StyleAppFrame.Render(inner)
}

// renderHeader — шапка: логотип слева, NETLINK-индикатор, [✕] справа.
func (m Model) renderHeader(width int) string {
	left := styles.StyleTitle.Render("◈ GOFER")
	netlink := m.renderNetlinkIndicator()
	right := styles.StyleButtonClose.Render("[✕]")

	contentWidth := width - 2 // Padding(0, 1)

	leftW := lipgloss.Width(left)
	netW := lipgloss.Width(netlink)
	rightW := lipgloss.Width(right)

	const gap2 = 3

	gap1 := contentWidth - leftW - netW - rightW - gap2
	if gap1 < 1 {
		gap1 = 1
	}

	content := left +
		strings.Repeat(" ", gap1) +
		netlink +
		strings.Repeat(" ", gap2) +
		right

	// Хитбокс [✕].
	closeY := 1
	closeX2 := m.width - 2
	closeX1 := closeX2 - rightW + 1
	*m.hitboxes = append(*m.hitboxes, screen.Hitbox{
		X1: closeX1, Y1: closeY,
		X2: closeX2, Y2: closeY,
		ID: "close",
	})

	return styles.StyleHeader.Width(width).Render(content)
}

// renderNetlinkIndicator — индикатор состояния соединения.
//
//	unknown  → серый   "[◌ NETLINK ...]"
//	online   → cyan    "[● NETLINK ON]"
//	offline  → red     "[○ NETLINK OFF]"
func (m Model) renderNetlinkIndicator() string {
	switch m.netlink {
	case netlinkOnline:
		return styles.StyleButtonNetOn.Render("[● NETLINK ON]")
	case netlinkOffline:
		return styles.StyleButtonNetOff.Render("[○ NETLINK OFF]")
	default:
		return styles.StyleFaint.Render("[◌ NETLINK ...]")
	}
}

// renderFooter — статус пользователя слева.
//
//	До логина:   "◈ не авторизован"
//	После:       "◈ {username}    {uuid_short}..."
func (m Model) renderFooter(width int) string {
	var content string
	if m.auth.IsAuthenticated() {
		uuidShort := m.auth.UserID
		if len(uuidShort) > 13 {
			uuidShort = uuidShort[:13] + "..."
		}

		// Состояние клика: вместо короткого UUID показываем фидбек.
		var rightText string
		switch m.copiedTarget {
		case "footer_uuid":
			rightText = styles.StyleOK.Render("✓ copied")
		case "fail:footer_uuid":
			rightText = styles.StyleDanger.Render("⚠ unavailable")
		default:
			rightText = styles.StyleFaint.Render(uuidShort)
		}

		left := styles.StyleAccent.Render("◈ " + m.auth.Username)
		innerW := width - 2
		content = spaceBetween(left, rightText, innerW)

		// Hitbox на правую часть футера (всегда там, где сейчас текст).
		// Y футера: рамка(1) + header-content(1) + header-border(1) + body + footer-border(1) ?
		// Проще — посчитать от конца окна.
		footerY := m.height - 2 // одну строку отнимаем за нижнюю рамку AppFrame
		rightW := lipgloss.Width(rightText)
		rightX2 := m.width - 2 // одна ячейка справа — рамка AppFrame
		rightX1 := rightX2 - rightW + 1
		*m.hitboxes = append(*m.hitboxes, screen.Hitbox{
			X1: rightX1, Y1: footerY,
			X2: rightX2, Y2: footerY,
			ID: "footer_uuid",
		})
	} else {
		content = styles.StyleFaint.Render("◈ не авторизован")
	}
	return styles.StyleFooter.Width(width).Render(content)
}

// renderBody делегирует экрану, собирает его хитбоксы в общий список.
func (m Model) renderBody(width, height int) string {
	originX := 1 // AppFrame съедает 1 ячейку слева
	originY := 3 // AppFrame(1) + Header-content(1) + Header-border(1)

	m.current.SetSize(width, height)
	m.current.SetOrigin(originX, originY)
	bodyContent := m.current.View()

	*m.hitboxes = append(*m.hitboxes, m.current.Hitboxes()...)

	return lipgloss.NewStyle().Width(width).Height(height).Render(bodyContent)
}
