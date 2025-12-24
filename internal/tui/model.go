package tui

import (
	"github.com/Microindole/quell/internal/core"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	list    list.Model
	svc     *core.Service
	keys    KeyMap // 🟢 新增：持有快捷键配置
	loading bool
	status  string
}

func NewModel(svc *core.Service) Model {
	var items []list.Item

	// 初始化列表
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Quell - Process Killer"
	l.SetShowHelp(false) // 禁用 list 自带的帮助，我们自己控制

	// 初始化快捷键
	keys := DefaultKeyMap()

	return Model{
		list:    l,
		svc:     svc,
		keys:    keys,
		loading: true,
		status:  "Scanning ports...",
	}
}

type processKilledMsg struct{ err error }

func (m Model) Init() tea.Cmd {
	// 启动时刷新列表
	return m.refreshListCmd()
}

// 辅助函数：刷新列表的 Cmd
func (m Model) refreshListCmd() tea.Cmd {
	return func() tea.Msg {
		// 调用 Service 获取数据
		procs, err := m.svc.GetProcesses()
		if err != nil {
			return nil // 或者返回一个 errMsg
		}

		// 将 core.Process 转换为 list.Item 接口
		items := make([]list.Item, len(procs))
		for i, p := range procs {
			items[i] = p
		}
		return items
	}
}

// 辅助函数：杀进程的 Cmd
func (m Model) killProcessCmd(pid int32) tea.Cmd {
	return func() tea.Msg {
		// 调用 Service 杀进程
		err := m.svc.Kill(pid)
		return processKilledMsg{err: err}
	}
}
