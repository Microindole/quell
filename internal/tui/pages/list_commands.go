package pages

import (
	"fmt"

	"github.com/Microindole/quell/internal/core"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// GetDefaultListActions 返回 ListView 的所有快捷键绑定
// 这样 ListView 就不需要知道具体的按键逻辑，只需要注册这些 Handler 即可
func GetDefaultListActions(v *ListView) []KeyHandler {
	return []KeyHandler{
		// 1. 进入详情页 (Enter/Space)
		{
			Binding: key.NewBinding(key.WithKeys("enter", "space"), key.WithHelp("enter", "detail")),
			Action: func(m View) (tea.Cmd, bool) {
				if i := v.list.SelectedItem(); i != nil {
					p := i.(core.Process)
					return Push(NewDetailView(&p, v.state)), true
				}
				return nil, false
			},
		},
		// 2. 切换排序 (Tab)
		{
			Binding: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "sort")),
			Action: func(m View) (tea.Cmd, bool) {
				v.currentSortIdx = (v.currentSortIdx + 1) % len(v.sorters)
				items := v.list.Items()
				v.list.SetItems(v.sortItems(items))
				v.status = fmt.Sprintf("Sorted by %s", v.sorters[v.currentSortIdx].Name())
				return nil, true
			},
		},
		// 3. 切换树状图 (t)
		{
			Binding: key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "tree view")),
			Action: func(m View) (tea.Cmd, bool) {
				v.treeMode = !v.treeMode
				return v.refreshListCmd(), true
			},
		},
		// 4. 普通杀进程 (x)
		{
			Binding: key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "kill")),
			Action:  makeKillAction(v, false),
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
				if i := v.list.SelectedItem(); i != nil {
					p := i.(core.Process)
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
				if i := v.list.SelectedItem(); i != nil {
					p := i.(core.Process)
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
		{
			Binding: key.NewBinding(key.WithKeys("esc", "q"), key.WithHelp("esc/q", "quit")), // 绑定 esc 和 q
			Action: func(m View) (tea.Cmd, bool) {
				// 弹出确认框
				return Push(NewConfirmDialog("Quit application?", tea.Quit)), true
			},
		},
	}
}

// 辅助函数：生成杀进程的 Action，避免重复代码
func makeKillAction(v *ListView, force bool) ActionFunc {
	return func(m View) (tea.Cmd, bool) {
		if i := v.list.SelectedItem(); i != nil {
			p := i.(core.Process)
			title := fmt.Sprintf("Sure to kill %s?", p.Name)
			if force {
				title = fmt.Sprintf("Sure to FORCE KILL %s?", p.Name)
			}
			// 复用 ListView 内部的 killCmd
			return Push(NewConfirmDialog(title, v.killCmd(p.PID, force))), true
		}
		return nil, false
	}
}
