package tui

import (
	"fmt"

	"github.com/Microindole/quell/internal/core"
	"github.com/Microindole/quell/internal/system"
	"github.com/Microindole/quell/internal/tui/commands"
	"github.com/Microindole/quell/internal/tui/components"
	"github.com/Microindole/quell/internal/tui/pages"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var appStyle = lipgloss.NewStyle().Padding(1, 2)

type Model struct {
	shared *pages.SharedState
	stack  []pages.View
	active pages.View
}

func NewModel(svc *core.Service) *Model {
	commands.RegisterAll(pages.CommandRegistry)
	
	state := &pages.SharedState{
		Service: svc,
		IsAdmin: system.IsAdmin(),
	}
	initialView := pages.NewListView(state)
	return &Model{
		shared: state,
		stack:  []pages.View{initialView},
		active: initialView,
	}
}

func (m *Model) Init() tea.Cmd {
	// 🔥 启动时，同时初始化页面 AND 启动全局心跳
	return tea.Batch(m.active.Init(), pages.TickCmd())
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case pages.PushViewMsg:
		m.stack = append(m.stack, msg.View)
		m.active = msg.View
		return m, msg.View.Init()

	case pages.PopViewMsg:
		if len(m.stack) > 1 {
			m.stack = m.stack[:len(m.stack)-1]
			m.active = m.stack[len(m.stack)-1]
		}
		return m, nil

	case pages.ReplaceViewMsg:
		if len(m.stack) > 0 {
			// 替换栈顶元素
			m.stack[len(m.stack)-1] = msg.View
			m.active = msg.View
			return m, msg.View.Init()
		}

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case pages.TickMsg:
		// 1. 续订下一个心跳 (保证循环不断)
		cmds = append(cmds, pages.TickCmd())
		// 2. 继续向下传递 msg，让 Active View 也有机会处理 Tick (比如刷新数据)
	}

	// 路由分发
	var cmd tea.Cmd
	m.active, cmd = m.active.Update(msg)
	cmds = append(cmds, cmd)

	// 更新栈顶
	if len(m.stack) > 0 {
		m.stack[len(m.stack)-1] = m.active
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) View() string {
	content := m.active.View()

	authIcon := "👤 User"
	if m.shared.IsAdmin {
		authIcon = "⚡ Admin"
	}

	extraInfo := ""
	// 如果是 ListView，显示特定状态
	if lv, ok := m.active.(*pages.ListView); ok {
		extraInfo = fmt.Sprintf(" | %s | Sort: %s", lv.GetStatus(), lv.GetSortName())
	}
	// 如果是 DetailView，也可以显示特定状态
	if _, ok := m.active.(*pages.DetailView); ok {
		extraInfo = " | Inspecting..."
	}

	statusText := authIcon + extraInfo
	statusBar := components.RenderStatusBar(statusText)

	return appStyle.Render(content + "\n" + statusBar)
}
