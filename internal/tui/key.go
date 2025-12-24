package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap 定义了所有可用的快捷键
type KeyMap struct {
	Up   key.Binding
	Down key.Binding
	Kill key.Binding
	Quit key.Binding
	Help key.Binding
}

// DefaultKeyMap 返回默认的快捷键设置
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Kill: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "kill process"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
	}
}

// ShortHelp 实现 help.KeyMap 接口 (显示在底部的简略帮助)
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Kill, k.Quit, k.Help}
}

// FullHelp 实现 help.KeyMap 接口 (展开后的完整帮助)
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Kill},
		{k.Quit, k.Help},
	}
}
