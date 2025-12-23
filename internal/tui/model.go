package tui

import (
	"github.com/Microindole/quell/internal/sys"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	list    list.Model
	loading bool
	err     error
}

func NewModel() Model {
	// 初始化列表
	items := []list.Item{} // 初始为空

	// 设置列表代理 (默认宽高，稍后会自适应)
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Quell - Processes"

	return Model{
		list:    l,
		loading: true,
	}
}

// Init: 启动时加载数据
func (m Model) Init() tea.Cmd {
	// 这是一个异步指令，去调用 sys 层获取数据
	return func() tea.Msg {
		procs, err := sys.GetProcesses()
		if err != nil {
			return nil // 简单处理
		}
		// 把 domain.Process 转换为 list.Item
		items := make([]list.Item, len(procs))
		for i, p := range procs {
			items[i] = p
		}
		return items // 发送数据消息
	}
}

// View 和 Update 稍后拆分，现在先简单写在一起测试
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height)

	case []list.Item: // 接收到 Init 返回的数据
		cmd := m.list.SetItems(msg)
		m.loading = false
		return m, cmd
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	return m.list.View()
}
