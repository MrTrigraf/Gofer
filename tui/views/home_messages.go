package views

import "github.com/gofer/tui/api"

// Сообщения загрузки списков. Приходят от команд из home_commands.go.
//
// ВАЖНО: все типы из этого файла перечислены в перехвате попапа
// (home_update.go). Попап модален для ввода, но фоновые ответы сервера
// должны проходить мимо него — иначе попап, открытый во время загрузки,
// съест ответ и списки навсегда останутся в состоянии loading.
// Добавил новый тип сюда — впиши его и туда.

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
