package pages

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type ActionFunc func(m View) (tea.Cmd, bool) // 注意：这里参数变为了 View 接口

type KeyHandler struct {
	Binding key.Binding
	Action  ActionFunc
}

type HandlerRegistry struct {
	handlers []KeyHandler
}

func (r *HandlerRegistry) Register(k key.Binding, action ActionFunc) {
	r.handlers = append(r.handlers, KeyHandler{Binding: k, Action: action})
}

func (r *HandlerRegistry) Handle(msg tea.KeyMsg, v View) (tea.Cmd, bool) {
	for _, h := range r.handlers {
		if key.Matches(msg, h.Binding) {
			return h.Action(v)
		}
	}
	return nil, false
}

func (r *HandlerRegistry) MakeHelp() []key.Binding {
	var bindings []key.Binding
	for _, h := range r.handlers {
		bindings = append(bindings, h.Binding)
	}
	return bindings
}
