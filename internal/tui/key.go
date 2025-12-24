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
	Sort      key.Binding
}

// ShortHelp 底部简略帮助
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Kill, k.Detail, k.Quit, k.Help}
}

// FullHelp 完整帮助
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Detail, k.Sort},
		{k.Kill, k.ForceKill, k.Back, k.Quit},
	}
}
