package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	minWidth  = 60
	minHeight = 20
)

func spaceBetween(left, right string, width int) string {
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}
