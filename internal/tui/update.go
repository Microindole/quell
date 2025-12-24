package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width-4, msg.Height-4)

	case tea.KeyMsg:
		// 1. 优先给 list 处理过滤
		if m.list.FilterState() == list.Filtering {
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}

		// 2. 委托给 Registry 处理所有注册过的按键
		// 只要 registry 找到了对应的处理函数，就执行并返回
		if cmd, handled := m.registry.Handle(msg, &m); handled {
			return m, cmd
		}
		// (如果需要 Up/Down 等原生 List 导航键，它们会在这里 fallthrough 到最后交给 list.Update)

	case tickMsg:
		return m, tea.Batch(m.refreshListCmd(), m.tickCmd())

	case []list.Item:
		// 数据加载逻辑保持不变，但使用 m.sortItems 策略方法
		sortedItems := m.sortItems(msg)
		cmd = m.list.SetItems(sortedItems)
		m.loading = false
		m.status = fmt.Sprintf("Scanned %d processes.", len(msg))
		// ... (保持原有的选中态刷新逻辑) ...
		return m, cmd

	case processKilledMsg:
		// ... (保持不变) ...
		if msg.err != nil {
			m.status = fmt.Sprintf("Error: %v", msg.err)
		} else {
			m.status = "Killed."
			m.inspecting = false
			m.list.RemoveItem(m.list.Index())
			return m, m.delayedRefreshCmd()
		}

	case delayedRefreshMsg:
		return m, m.refreshListCmd()
	}

	// 兜底：如果没处于详情模式，且上面的 KeyMsg 没拦截，交给 list 组件 (处理 j/k/up/down)
	if !m.inspecting {
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}
