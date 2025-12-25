package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// 普通状态：紫色
	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	// 错误状态：红色
	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#FF0000")).
			Padding(0, 1)

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1A1A1A")).
			Background(lipgloss.Color("#FFA500")).
			Padding(0, 1).
			Bold(true)
)

// RenderStatusBar 渲染底部状态栏
func RenderStatusBar(status string) string {
	if status == "" {
		return ""
	}

	// 1. 优先判断 Error
	if strings.HasPrefix(status, "Error") {
		return errorStyle.Render(status)
	}

	// 2. 判断警告/确认状态 (修复了 status[0] == '⚠️' 的报错)
	// 使用 HasPrefix 可以正确处理多字节字符
	if strings.HasPrefix(status, "⚠️") || strings.HasPrefix(status, "?") {
		return warningStyle.Render(status)
	}

	// 3. 默认状态
	return statusStyle.Render(status)
}
