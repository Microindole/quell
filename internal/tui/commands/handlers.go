package commands

import (
	"github.com/Microindole/quell/internal/tui/pages"
	tea "github.com/charmbracelet/bubbletea"
)

// HelpCmd 显示帮助页面
func HelpCmd(args []string, state *pages.SharedState) (pages.View, tea.Cmd) {
	// 引用 pages 包创建视图
	return pages.NewHelpView(), nil
}

// QuitCmd 退出程序
func QuitCmd(args []string, state *pages.SharedState) (pages.View, tea.Cmd) {
	return nil, tea.Quit
}

// 这里以后可以扩展更多命令，例如:
// func TreeCmd(args []string, state *pages.SharedState) (pages.View, tea.Cmd) { ... }
