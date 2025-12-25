package tui

import (
	"fmt"

	"github.com/Microindole/quell/internal/config"
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

func NewModel(svc *core.Service, cfg *config.Config) *Model {
	commands.RegisterAll(pages.CommandRegistry)
	state := &pages.SharedState{
		Service: svc,
		IsAdmin: system.IsAdmin(),
	}
	initialView := pages.NewListView(state, cfg.SortIndex, cfg.TreeMode)
	return &Model{
		shared: state,
		stack:  []pages.View{initialView},
		active: initialView,
	}
}

// GetSnapshot 收集当前应用状态用于保存
func (m *Model) GetSnapshot() *config.Config {
	cfg := &config.Config{}

	// 1. 获取 Service 中的暂停列表 (返回的是匿名结构体切片)
	rawList := m.shared.Service.GetPausedProcs()

	// 2. 转换为 config 包需要的结构体
	var pausedProcs []config.PausedProcess
	for _, item := range rawList {
		pausedProcs = append(pausedProcs, config.PausedProcess{
			PID:        item.PID,
			CreateTime: item.CreateTime,
		})
	}
	cfg.PausedProcs = pausedProcs

	// 3. 获取 ListView 的状态
	if len(m.stack) > 0 {
		if lv, ok := m.stack[0].(*pages.ListView); ok {
			sortIdx, treeMode := lv.GetState()
			cfg.SortIndex = sortIdx
			cfg.TreeMode = treeMode
		}
	}

	return cfg
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
			if _, ok := m.active.(*pages.ConfirmDialog); ok {
				return m, tea.Quit
			}
			return m, pages.Push(pages.NewConfirmDialog("Really quit Quell?", tea.Quit))
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
