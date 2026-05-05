package tui

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	if m.width < minWidth || m.height < minHeight {
		return m.renderTooSmall()
	}

	return m.renderApp()
}
