package main

import (
	"io"
	"log"
	"log/slog"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gofer/tui"
	"github.com/gofer/tui/api"
	"github.com/gofer/tui/serverstore"
)

func main() {

	// При необходимости лог включается заменой io.Discard на файл/os.Stderr.
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))

	store, err := serverstore.Load()
	if err != nil {
		slog.Error("failed to load server store", "err", err)
		os.Exit(1)
	}
	slog.Info("server store loaded",
		"count", len(store.Servers), "selected", store.Selected)

	client := api.New(store.Selected)

	p := tea.NewProgram(
		tui.New(client, store),
		tea.WithMouseCellMotion(),
		tea.WithAltScreen(),
	)
	if _, err := p.Run(); err != nil {
		log.Fatalf("tui error: %v", err)
	}
}
