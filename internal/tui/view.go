package tui

import (
	"github.com/Microindole/quell/internal/tui/components"
	"github.com/charmbracelet/lipgloss"
)

var appStyle = lipgloss.NewStyle().Padding(1, 2)

func (m Model) View() string {
	if m.loading {
		return appStyle.Render("Loading processes...")
	}

	// 1. 渲染列表
	listView := m.list.View()

	// 2. 渲染状态栏 (使用新组件)
	statusBar := components.RenderStatusBar(m.status)

	// 3. 组合
	return appStyle.Render(listView + "\n" + statusBar)
}
