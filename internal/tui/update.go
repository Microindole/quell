package tui

import (
	"fmt"

	"github.com/Microindole/quell/internal/domain"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// 1. 窗口大小改变 (全屏自适应)
	case tea.WindowSizeMsg:
		h, v := appStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)

	// 2. 按键事件
	case tea.KeyMsg:
		// 如果列表正在过滤搜索中，不要拦截按键，交给 list 自己处理
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "x": // ⚔️ 按 X 杀进程
			if selectedItem := m.list.SelectedItem(); selectedItem != nil {
				p := selectedItem.(domain.Process)
				m.status = fmt.Sprintf("Killing %s (PID: %d)...", p.Name, p.PID)
				// 返回一个指令：去杀进程
				return m, killProcess(p.PID)
			}
		}

	// 3. 列表数据加载完成
	case []list.Item:
		cmd := m.list.SetItems(msg)
		m.loading = false
		m.status = fmt.Sprintf("Scanned %d processes.", len(msg))
		return m, cmd

	// 4. 进程杀完了 (收到结果)
	case processKilledMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Error: %v", msg.err)
		} else {
			m.status = "Killed successfully! Refreshing..."
			// 杀成功后，立即触发刷新列表
			return m, refreshList
		}
	}

	// 更新列表组件本身
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}
