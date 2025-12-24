package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap 定义了所有可用的快捷键
type KeyMap struct {
	Up        key.Binding
	Down      key.Binding
	Kill      key.Binding
	ForceKill key.Binding
	Quit      key.Binding
	Help      key.Binding
	Detail    key.Binding // Enter 查看详情
	Back      key.Binding // Esc 返回列表
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
			key.WithHelp("x", "terminate"),
		),
		ForceKill: key.NewBinding(
			key.WithKeys("X"),
			key.WithHelp("X", "force kill"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c", "q"),
			key.WithHelp("q/ctrl+c", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		// 👇 新增定义
		Detail: key.NewBinding(
			key.WithKeys("enter", "space"),
			key.WithHelp("enter", "view details"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc", "backspace"),
			key.WithHelp("esc", "back"),
		),
	}
}

// ShortHelp 底部简略帮助
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Kill, k.Detail, k.Quit, k.Help}
}

// FullHelp 完整帮助
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Detail},
		{k.Kill, k.ForceKill, k.Back, k.Quit},
	}
}
