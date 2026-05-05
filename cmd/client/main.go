package main

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gofer/tui"
	"github.com/gofer/tui/api"
)

func main() {
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
