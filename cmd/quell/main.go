package main

import (
	"fmt"
	"os"

	"github.com/Microindole/quell/internal/core"
	"github.com/Microindole/quell/internal/system"
	"github.com/Microindole/quell/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	sysProvider := system.NewLocalProvider()
	svc := core.NewService(sysProvider)
	model := tui.NewModel(svc)

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error starting Quell: %v", err)
		os.Exit(1)
	}
}
