package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gofer/tui/api"
)

// Период пинга сервера. 5 секунд — хороший компромисс:
// для юзера почти моментально, для сервера — нулевая нагрузка.
const netlinkPingInterval = 5 * time.Second

// netlinkPingMsg — тик от таймера. Реакция: послать пинг.
type netlinkPingMsg struct{}

// netlinkResultMsg — результат пинга. Обновляет m.netlink.
type netlinkResultMsg struct {
	online bool
}

// pingCmd — выполняет один HTTP-запрос к /health и возвращает результат
// как netlinkResultMsg.
func pingCmd(client *api.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		err := client.Health(ctx)
		return netlinkResultMsg{online: err == nil}
	}
}

// scheduleNextPing — возвращает команду "через netlinkPingInterval
// пришли netlinkPingMsg". Используется для зацикленной работы таймера.
func scheduleNextPing() tea.Cmd {
	return tea.Tick(netlinkPingInterval, func(time.Time) tea.Msg {
		return netlinkPingMsg{}
	})
}
