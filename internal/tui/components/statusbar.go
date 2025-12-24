package components

import "github.com/charmbracelet/lipgloss"

var (
	// 定义样式
	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#FF0000")).
			Padding(0, 1)
)

// RenderStatusBar 渲染底部状态栏
func RenderStatusBar(status string) string {
	if status == "" {
		return ""
	}

	// 简单判断：如果是 Error 开头，显示红色，否则显示紫色
	if len(status) >= 5 && status[:5] == "Error" {
		return errorStyle.Render(status)
	}
	return statusStyle.Render(status)
}
