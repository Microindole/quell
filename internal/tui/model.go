package tui

import (
	"time"

	"github.com/Microindole/quell/internal/core"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

const heartbeatInterval = 2 * time.Second

type tickMsg time.Time

type Model struct {
	list    list.Model
	svc     *core.Service
	keys    KeyMap
	loading bool
	status  string
	// 👇 新增状态
	inspecting bool          // 是否处于详情模式
	selected   *core.Process // 当前正在查看的进程
}

func NewModel(svc *core.Service) Model {
	items := []list.Item{}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Quell - Process Killer"
	l.SetShowHelp(false) // 禁用自带帮助，使用我们自己的状态栏

	keys := DefaultKeyMap()

	return Model{
		list:       l,
		svc:        svc,
		keys:       keys,
		loading:    true,
		status:     "Scanning ports...",
		inspecting: false,
	}
}

type delayedRefreshMsg struct{}

type processKilledMsg struct{ err error }

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.refreshListCmd(),
		m.tickCmd(),
	)
}

func (m Model) refreshListCmd() tea.Cmd {
	return func() tea.Msg {
		procs, err := m.svc.GetProcesses()
		if err != nil {
			return nil
		}
		items := make([]list.Item, len(procs))
		for i, p := range procs {
			items[i] = p
		}
		return items
	}
}

func (m Model) killProcessCmd(pid int32, force bool) tea.Cmd {
	return func() tea.Msg {
		err := m.svc.Kill(pid, force)
		return processKilledMsg{err: err}
	}
}

func (m Model) delayedRefreshCmd() tea.Cmd {
	return tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
		return delayedRefreshMsg{}
	})
}

func (m Model) getTarget() (int32, string) {
	if m.inspecting && m.selected != nil {
		return m.selected.PID, m.selected.Name
	}
	if i := m.list.SelectedItem(); i != nil {
		p := i.(core.Process) // 类型断言
		return p.PID, p.Name
	}
	return 0, ""
}

func (m Model) tickCmd() tea.Cmd {
	return tea.Tick(heartbeatInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
