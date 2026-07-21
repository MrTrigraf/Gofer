package views

import "github.com/gofer/tui/api"

// Сообщения загрузки списков. Приходят от команд из home_commands.go.
//
// Это фоновые ответы сервера. Они проходят мимо активного попапа
// автоматически: попап перехватывает только пользовательский ввод
// (см. isInputMsg в home_update.go), поэтому синхронизировать никакие
// списки при добавлении нового типа больше не нужно.

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
