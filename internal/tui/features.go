package tui

import (
	"fmt"

	"github.com/Microindole/quell/internal/core"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// 注意：这里去掉了 (m *Model) 接收者，变成了普通函数
func registerCoreActions(m *Model) {
	// Quit
	m.registry.Register(key.NewBinding(key.WithKeys("ctrl+c", "q"), key.WithHelp("q", "quit")),
		func(current *Model) (tea.Cmd, bool) {
			return tea.Quit, true
		})

	// Enter Detail
	m.registry.Register(key.NewBinding(key.WithKeys("enter", "space"), key.WithHelp("enter", "details")),
		func(current *Model) (tea.Cmd, bool) {
			if !current.inspecting {
				if i := current.list.SelectedItem(); i != nil {
					p := i.(core.Process)
					current.selected = &p
					current.inspecting = true
					current.status = fmt.Sprintf("Inspecting PID %d...", p.PID)
					return nil, true
				}
			}
			return nil, false
		})

	// Back
	m.registry.Register(key.NewBinding(key.WithKeys("esc", "backspace"), key.WithHelp("esc", "back")),
		func(current *Model) (tea.Cmd, bool) {
			if current.inspecting {
				current.inspecting = false
				current.status = "Back to list."
				return nil, true
			}
			return nil, false
		})

	// Kill
	m.registry.Register(key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "terminate")),
		func(current *Model) (tea.Cmd, bool) {
			if pid, name := current.getTarget(); pid != 0 {
				current.status = fmt.Sprintf("Terminating %s...", name)
				return current.killProcessCmd(pid, false), true
			}
			return nil, false
		})
}

// 注意：这里也变成了普通函数
func registerSortActions(m *Model) {
	m.registry.Register(key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "toggle sort")),
		func(current *Model) (tea.Cmd, bool) {
			current.currentSortIdx = (current.currentSortIdx + 1) % len(current.sorters)
			items := current.list.Items()
			sortedItems := current.sortItems(items)
			cmd := current.list.SetItems(sortedItems)
			current.status = fmt.Sprintf("Sorted by %s", current.sorters[current.currentSortIdx].Name())
			return cmd, true
		})
}
