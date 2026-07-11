package clipboard

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type CopiedMsg struct {
	Target string
}

type CopyFailedMsg struct {
	Target string
	Err    error
}

type ClearCopiedMsg struct {
	Target string
}

type UnavailableMsg struct {
	Err error
}

const feedbackTimeout = 1500 * time.Millisecond

func CopyCmd(target, text string) tea.Cmd {
	return func() tea.Msg {
		if err := Copy(text); err != nil {
			return CopyFailedMsg{Target: target, Err: err}
		}
		return CopiedMsg{Target: target}
	}
}

func ClearAfterTimeout(target string) tea.Cmd {
	return tea.Tick(feedbackTimeout, func(time.Time) tea.Msg {
		return ClearCopiedMsg{Target: target}
	})
}

func CheckCmd() tea.Cmd {
	return func() tea.Msg {
		if err := Available(); err != nil {
			return UnavailableMsg{Err: err}
		}
		return nil
	}
}
