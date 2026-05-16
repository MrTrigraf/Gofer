package main

import (
	"log"
	"log/slog"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gofer/tui"
	"github.com/gofer/tui/api"
)

func main() {
	// TEST(8.4.1.a): направляем slog в файл, чтобы логи не ломали TUI.
	// Открой второй терминал и запусти: tail -f client.log
	logFile, err := os.OpenFile("client.log",
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("open log file: %v", err)
	}
	defer logFile.Close()
	slog.SetDefault(slog.New(slog.NewTextHandler(logFile, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
	slog.Info("=== client started ===")

	// HTTP-клиент к серверу. Пока адрес жёстко прописан —
	// в будущем вынесем в конфиг.
	client := api.New("http://localhost:8080")

	p := tea.NewProgram(
		tui.New(client),
		tea.WithMouseCellMotion(),
		tea.WithAltScreen(),
	)
	if _, err := p.Run(); err != nil {
		log.Fatalf("tui error: %v", err)
	}
}
