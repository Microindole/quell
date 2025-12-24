package tui

import (
	"fmt"

	"github.com/Microindole/quell/internal/core"
	"github.com/charmbracelet/bubbles/key"
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
		// 如果在列表过滤模式，优先交给 list 处理
		if m.list.FilterState() == list.Filtering {
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}

		switch {
		// 1. 全局退出
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		// 2. 返回键 (从详情页返回)
		case key.Matches(msg, m.keys.Back):
			if m.inspecting {
				m.inspecting = false
				m.status = "Back to list."
				return m, nil
			}

		// 3. 进入详情页 (Enter)
		case key.Matches(msg, m.keys.Detail):
			if !m.inspecting {
				if i := m.list.SelectedItem(); i != nil {
					p := i.(core.Process)
					m.selected = &p // 记录当前选中的进程
					m.inspecting = true
					m.status = fmt.Sprintf("Inspecting PID %d...", p.PID)
					return m, nil
				}
			}

		// 4. 优雅退出 (x)
		case key.Matches(msg, m.keys.Kill):
			if pid, name := m.getTarget(); pid != 0 {
				m.status = fmt.Sprintf("Terminating %s (PID: %d)...", name, pid)
				// force = false
				return m, m.killProcessCmd(pid, false)
			}

		// 5. 强制击杀 (X)
		case key.Matches(msg, m.keys.ForceKill):
			if pid, name := m.getTarget(); pid != 0 {
				m.status = fmt.Sprintf("KILLING %s (PID: %d)!!!", name, pid)
				// force = true
				return m, m.killProcessCmd(pid, true)
			}
		}
	case tickMsg:
		return m, tea.Batch(
			m.refreshListCmd(), // 获取最新内存/新进程
			m.tickCmd(),        // 预约下一次心跳
		)
	// 数据加载回调
	case []list.Item:
		cmd = m.list.SetItems(msg)
		m.loading = false
		m.status = fmt.Sprintf("Scanned %d processes.", len(msg))

		if m.inspecting && m.selected != nil {
			found := false
			for _, item := range msg {
				p := item.(core.Process)
				// 找到当前正在看的那个 PID，更新数据
				if p.PID == m.selected.PID {
					m.selected = &p // 指针指向最新的 Process 结构体（包含新 CPU/Mem）
					found = true
					break
				}
			}
			// 如果新列表里找不到这个进程了（说明它刚才退出了）
			if !found {
				m.inspecting = false // 强制退出详情页
				m.status = "Process terminated/closed port."
			}
		}

		return m, cmd

	// 杀进程回调
	case processKilledMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Error: %v", msg.err)
		} else {
			m.status = "Killed."
			m.inspecting = false
			index := m.list.Index()
			m.list.RemoveItem(index)
			return m, m.delayedRefreshCmd()
		}

	case delayedRefreshMsg:
		return m, m.refreshListCmd()
	}

	if !m.inspecting {
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}
