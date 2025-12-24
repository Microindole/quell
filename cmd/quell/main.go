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
	// 1. 实例化底层系统实现 (System Layer)
	sysProvider := system.NewLocalProvider()

	// 2. 实例化核心服务 (Core Layer)
	svc := core.NewService(sysProvider)

	// 3. 实例化界面并注入服务 (TUI Layer)
	model := tui.NewModel(svc)

	// 4. 启动程序
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error starting Quell: %v", err)
		os.Exit(1)
	}
}
