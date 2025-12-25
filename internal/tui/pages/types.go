package pages

import (
	"time"

	"github.com/Microindole/quell/internal/core"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// View 定义页面组件的通用接口
type View interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (View, tea.Cmd)
	View() string
	ShortHelp() []key.Binding
}

// SharedState 存放全局共享状态
type SharedState struct {
	Service *core.Service
	IsAdmin bool
}

type PushViewMsg struct{ View View }
type PopViewMsg struct{}
type ReplaceViewMsg struct{ View View }

func Push(v View) tea.Cmd {
	return func() tea.Msg { return PushViewMsg{View: v} }
}
func Pop() tea.Cmd {
	return func() tea.Msg { return PopViewMsg{} }
}
func Replace(v View) tea.Cmd { // 🔥 新增
	return func() tea.Msg { return ReplaceViewMsg{View: v} }
}

// TickMsg 全局心跳消息
type TickMsg time.Time

func TickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}
