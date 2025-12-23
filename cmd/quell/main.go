package main

import (
	"fmt"
	"os"

	"github.com/Microindole/quell/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	p := tea.NewProgram(tui.NewModel(), tea.WithAltScreen()) // WithAltScreen 开启全屏模式
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error starting Quell: %v", err)
		os.Exit(1)
	}
}
