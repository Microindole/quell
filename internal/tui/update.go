package tui

import (
	"fmt"

	"github.com/Microindole/quell/internal/core"
	"github.com/charmbracelet/bubbles/key" // 🟢 引入 key 包
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width-4, msg.Height-4)

	case tea.KeyMsg:
		// 如果在过滤输入中，交给 list 处理
		if m.list.FilterState() == list.Filtering {
			break
		}

		// 🟢 使用 KeyMap 匹配按键
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Kill):
			if selectedItem := m.list.SelectedItem(); selectedItem != nil {
				p := selectedItem.(core.Process)
				m.status = fmt.Sprintf("Killing %s (PID: %d)...", p.Name, p.PID)
				return m, m.killProcessCmd(p.PID)
			}
		}

	case []list.Item:
		cmd := m.list.SetItems(msg)
		m.loading = false
		m.status = fmt.Sprintf("Scanned %d processes.", len(msg))
		return m, cmd

	case processKilledMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Error: %v", msg.err)
		} else {
			m.status = "Killed successfully! Refreshing..."
			return m, m.refreshListCmd()
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}
