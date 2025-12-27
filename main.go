package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// 1. 定义 Model (状态)
type model struct {
	cursor   int
	choices  []string
	selected map[int]struct{}
}

// 2. 初始化数据
func initialModel() model {
	return model{
		choices:  []string{"Port 8080 (Java)", "Port 3000 (Node)", "Port 5432 (Postgres)"}, // 假数据，之后替换为真实进程
		selected: make(map[int]struct{}),
	}
}

// Init (初始化命令，比如启动时加载数据)
func (m model) Init() tea.Cmd {
	return nil
}

// Update (消息循环：处理按键)
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter", " ":
			_, ok := m.selected[m.cursor]
			if ok {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = struct{}{}
			}
		}
	}
	return m, nil
}

// View (渲染界面)
func (m model) View() string {
	// 定义一些简单的样式
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA")).Background(lipgloss.Color("#7D56F4")).Padding(0, 1)

	s := titleStyle.Render(" Quell - Process Killer ") + "\n\n"

	for i, choice := range m.choices {
		cursor := " " // 默认没选中
		if m.cursor == i {
			cursor = ">" // 当前光标位置
		}

		checked := " " // 未勾选
		if _, ok := m.selected[i]; ok {
			checked = "x" // 已勾选 (模拟 Kill 标记)
		}

		s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, choice)
	}

	s += "\nPress q to quit.\n"
	return s
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
