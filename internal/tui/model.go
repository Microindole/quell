package tui

import (
	"github.com/Microindole/quell/internal/sys"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	list    list.Model
	loading bool
	// 👇 新增：用于显示底部状态栏的信息
	status string
}

func NewModel() Model {
	items := []list.Item{}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Quell - Process Killer"

	// 设置左下角的帮助文本
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "kill process")),
		}
	}

	return Model{
		list:    l,
		loading: true,
		status:  "Scanning ports...", // 初始状态
	}
}

// 定义一个消息类型，告诉 Update 进程杀完了
type processKilledMsg struct{ err error }

// Init 保持不变
func (m Model) Init() tea.Cmd {
	return refreshList
}

// 辅助函数：刷新列表的指令
func refreshList() tea.Msg {
	procs, err := sys.GetProcesses()
	if err != nil {
		return nil
	}
	items := make([]list.Item, len(procs))
	for i, p := range procs {
		items[i] = p
	}
	return items
}

// 辅助函数：杀进程的指令
func killProcess(pid int32) tea.Cmd {
	return func() tea.Msg {
		err := sys.KillProcess(pid)
		return processKilledMsg{err: err}
	}
}
