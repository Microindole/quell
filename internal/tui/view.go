package tui

import (
	"fmt"
	"strings"

	"github.com/Microindole/quell/internal/tui/components"
	"github.com/charmbracelet/lipgloss"
)

var (
	appStyle = lipgloss.NewStyle().Padding(1, 2)

	// 详情页标题样式
	detailTitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1).
		Bold(true)

	// 详情页字段名样式
	labelStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7D56F4")).
		Bold(true).
		Width(10) // 固定宽度对齐

	// 详情内容框样式
	detailBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		Padding(1, 2).
		MarginTop(1)
)

func (m Model) View() string {
	if m.loading {
		return appStyle.Render("Loading processes...")
	}

	var content string

	if m.inspecting && m.selected != nil {
		content = m.renderDetailView()
	} else {
		content = m.list.View()
	}

	authIcon := "👤 User"
	if m.isAdmin {
		authIcon = "⚡ Admin"
	}

	statusText := fmt.Sprintf("%s | %s | Sort: %s", authIcon, m.status, m.getSortName())

	statusBar := components.RenderStatusBar(statusText)

	return appStyle.Render(content + "\n" + statusBar)
}

// 辅助函数：绘制详情卡片
func (m Model) renderDetailView() string {
	p := m.selected

	// 格式化内存
	memMB := float64(p.MemoryUsage) / 1024 / 1024

	// 构建字段行
	rows := []string{
		fmt.Sprintf("%s %s", labelStyle.Render("Name:"), p.Name),
		fmt.Sprintf("%s %d", labelStyle.Render("PID:"), p.PID),
		fmt.Sprintf("%s %d (%s)", labelStyle.Render("Port:"), p.Port, p.Protocol),
		fmt.Sprintf("%s %s", labelStyle.Render("User:"), p.User),
		"", // 空行
		fmt.Sprintf("%s %.1f%%", labelStyle.Render("CPU:"), p.CpuPercent),
		fmt.Sprintf("%s %.1f MB", labelStyle.Render("Memory:"), memMB),
		"", // 空行
		labelStyle.Render("Command:"),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#A0A0A0")).Render(p.Cmdline),
	}

	body := strings.Join(rows, "\n")

	// 组装标题和边框
	header := detailTitleStyle.Render(fmt.Sprintf(" Process Detail: %s ", p.Name))
	box := detailBoxStyle.Render(body)

	// 底部提示
	help := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render("\nPress [Esc] to back • [x] to kill")

	return header + "\n" + box + help
}
