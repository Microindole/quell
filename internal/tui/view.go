package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// 定义简单的样式
var (
	appStyle    = lipgloss.NewStyle().Padding(1, 2)
	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)
	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#FF0000")).
			Padding(0, 1)
)

func (m Model) View() string {
	if m.loading {
		return "Loading processes..."
	}

	// 渲染状态栏
	// 这里简单判断一下，如果 status 包含 "Error" 就用红色背景，否则紫色
	var statusBar string
	if len(m.status) > 0 {
		if len(m.status) >= 5 && m.status[:5] == "Error" {
			statusBar = errorStyle.Render(m.status)
		} else {
			statusBar = statusStyle.Render(m.status)
		}
	}

	// 组合：列表 + 换行 + 状态栏
	return appStyle.Render(m.list.View() + "\n" + statusBar)
}
