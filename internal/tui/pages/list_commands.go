package pages

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func GetDefaultListActions(v *ListView) []KeyHandler {
	return []KeyHandler{
		// 1. 进入详情页 (Enter)
		{
			Binding: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "detail")),
			Action: func(m View) (tea.Cmd, bool) {
				// 🔥 直接使用组件提供的安全方法
				if p := v.processList.SelectedItem(); p != nil {
					w := v.processList.Inner().Width() + 4
					if w < 10 {
						w = 80
					}
					return Push(NewDetailView(p, v.state, w)), true
				}
				return nil, false
			},
		},
		// 2. 切换排序 (Tab)
		{
			Binding: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "sort")),
			Action: func(m View) (tea.Cmd, bool) {
				v.currentSortIdx = (v.currentSortIdx + 1) % len(v.sorters)
				v.updateListItems()
				return nil, true
			},
		},
		// 3. 切换树状图 (t)
		{
			Binding: key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "tree")),
			Action: func(m View) (tea.Cmd, bool) {
				v.treeMode = !v.treeMode
				v.updateListItems()
				return nil, true
			},
		},
		// 4. 普通杀进程 (x)
		{
			Binding: key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "kill")),
			Action: func(m View) (tea.Cmd, bool) {
				// A. 批量处理
				if len(v.selectedPids) > 0 {
					count := len(v.selectedPids)
					msg := fmt.Sprintf("Kill %d selected processes?", count)
					var cmds []tea.Cmd
					for pid := range v.selectedPids {
						cmds = append(cmds, v.killCmd(pid, false))
					}
					cmds = append(cmds, func() tea.Msg { return ClearSelectionMsg{} })
					return Push(NewConfirmDialog(msg, tea.Batch(cmds...))), true
				}

				// B. 单个处理
				if p := v.processList.SelectedItem(); p != nil {
					return Push(NewConfirmDialog(
						fmt.Sprintf("Kill process %d (%s)?", p.PID, p.Name),
						v.killCmd(p.PID, false),
					)), true
				}
				return nil, false
			},
		},
		// 5. 强制杀进程 (X)
		{
			Binding: key.NewBinding(key.WithKeys("X"), key.WithHelp("X", "force kill")),
			Action:  makeKillAction(v, true),
		},
		// 6. 暂停进程 (s)
		{
			Binding: key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "suspend")),
			Action: func(m View) (tea.Cmd, bool) {
				if p := v.processList.SelectedItem(); p != nil {
					return func() tea.Msg {
						return ProcessActionMsg{Err: v.state.Service.Suspend(p.PID), Action: "Suspended"}
					}, true
				}
				return nil, false
			},
		},
		// 7. 恢复进程 (c)
		{
			Binding: key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "continue")),
			Action: func(m View) (tea.Cmd, bool) {
				if p := v.processList.SelectedItem(); p != nil {
					return func() tea.Msg {
						return ProcessActionMsg{Err: v.state.Service.Resume(p.PID), Action: "Resumed"}
					}, true
				}
				return nil, false
			},
		},
		// 8. 呼出命令输入框 (`)
		{
			Binding: key.NewBinding(key.WithKeys("`"), key.WithHelp("`", "command")),
			Action: func(m View) (tea.Cmd, bool) {
				return Push(NewCommandInput(v.state, "")), true
			},
		},
		// 9. 快速批量查杀 (P)
		{
			Binding: key.NewBinding(key.WithKeys("P"), key.WithHelp("P", "pkill")),
			Action: func(m View) (tea.Cmd, bool) {
				return Push(NewCommandInput(v.state, "/pkill ")), true
			},
		},
		// 10. 空格键多选
		{
			Binding: key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "select")),
			Action: func(m View) (tea.Cmd, bool) {
				if p := v.processList.SelectedItem(); p != nil {
					if v.selectedPids[p.PID] {
						delete(v.selectedPids, p.PID)
					} else {
						v.selectedPids[p.PID] = true
					}
					return v.updateListItems(), true
				}
				return nil, false
			},
		},
		// 11. 退出逻辑
		{
			Binding: key.NewBinding(key.WithKeys("esc", "q"), key.WithHelp("esc/q", "quit")),
			Action: func(m View) (tea.Cmd, bool) {
				if len(v.selectedPids) > 0 {
					v.selectedPids = make(map[int32]bool)
					return v.updateListItems(), true
				}
				return Push(NewConfirmDialog("Quit application?", tea.Quit)), true
			},
		},
	}
}

// 辅助函数 unwrapProcess 不再需要，可以删除

func makeKillAction(v *ListView, force bool) ActionFunc {
	return func(m View) (tea.Cmd, bool) {
		if p := v.processList.SelectedItem(); p != nil {
			title := fmt.Sprintf("Sure to kill %s?", p.Name)
			if force {
				title = fmt.Sprintf("Sure to FORCE KILL %s?", p.Name)
			}
			return Push(NewConfirmDialog(title, v.killCmd(p.PID, force))), true
		}
		return nil, false
	}
}
