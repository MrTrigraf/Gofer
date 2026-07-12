package views

import (
	"context"
	"errors"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gofer/tui/api"
)

// apiTimeout — общий таймаут на любой HTTP-запрос к серверу.
const apiTimeout = 5 * time.Second

// === КОМАНДЫ ===
//
// Каждая команда: дёрнуть API → вернуть Done- или Error-сообщение
// из home_messages.go. Модель они не трогают.

func loadChannelsCmd(client *api.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
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
		ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
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
		ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
		defer cancel()

		if err := client.LeaveChannel(ctx, channelID); err != nil {
			return LeaveErrorMsg{Err: err}
		}
		return LeaveDoneMsg{ChannelID: channelID}
	}
}

func createChannelCmd(client *api.Client, name string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
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
		ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
		defer cancel()

		if err := client.JoinChannel(ctx, channelID); err != nil {
			return JoinChannelErrorMsg{Err: err}
		}
		return JoinChannelDoneMsg{ChannelID: channelID}
	}
}

func deleteChannelCmd(client *api.Client, channelID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
		defer cancel()

		if err := client.DeleteChannel(ctx, channelID); err != nil {
			return DeleteErrorMsg{Err: err}
		}
		return DeleteDoneMsg{ChannelID: channelID}
	}
}

func deleteDMCmd(client *api.Client, dmID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
		defer cancel()

		if err := client.DeleteDM(ctx, dmID); err != nil {
			return DeleteDMErrorMsg{Err: err}
		}
		return DeleteDMDoneMsg{DMID: dmID}
	}
}

func startDMCmd(client *api.Client, userID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
		defer cancel()

		dm, err := client.StartDM(ctx, userID)
		if err != nil {
			return StartDMErrorMsg{Err: err}
		}
		return StartDMDoneMsg{DM: dm}
	}
}

// === ПЕРЕВОД ОШИБОК API В ТЕКСТ ДЛЯ ЮЗЕРА ===

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
