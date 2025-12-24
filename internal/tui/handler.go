package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// ActionFunc 是具体的业务逻辑函数签名
// 返回值: (cmd, handled) -> handled 为 true 表示该按键已被处理，不需要继续传递
type ActionFunc func(m *Model) (tea.Cmd, bool)

// KeyHandler 将按键定义与业务逻辑绑定
type KeyHandler struct {
	Binding key.Binding
	Action  ActionFunc
	Desc    string // 用于帮助菜单分类（可选）
}

// HandlerRegistry 用于管理所有的按键处理器
type HandlerRegistry struct {
	handlers []KeyHandler
}

func (r *HandlerRegistry) Register(k key.Binding, action ActionFunc) {
	r.handlers = append(r.handlers, KeyHandler{
		Binding: k,
		Action:  action,
	})
}

// Handle 尝试处理按键消息
func (r *HandlerRegistry) Handle(msg tea.KeyMsg, m *Model) (tea.Cmd, bool) {
	for _, h := range r.handlers {
		if key.Matches(msg, h.Binding) {
			return h.Action(m)
		}
	}
	return nil, false
}

// MakeHelp 生成帮助菜单所需的数据 (适配 bubbles/help)
func (r *HandlerRegistry) MakeHelp() []key.Binding {
	var bindings []key.Binding
	for _, h := range r.handlers {
		bindings = append(bindings, h.Binding)
	}
	return bindings
}
