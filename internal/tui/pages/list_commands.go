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
			Binding: key.NewBinding(key.WithKeys("enter", "right"), key.WithHelp("enter", "detail")),
			Action: func(m View) (tea.Cmd, bool) {
				if i := v.list.SelectedItem(); i != nil {
					var p core.Process
					if sp, ok := i.(SelectableProcess); ok {
						p = sp.Process
					} else if raw, ok := i.(core.Process); ok {
						p = raw
					}

					// 获取宽度 (处理上一轮提到的逻辑)
					w := v.list.Width() + 4
					if w < 10 {
						w = 80
					}
					return Push(NewDetailView(&p, v.state, w)), true
				}
				return nil, false
			},
		},
		// 2. 切换排序 (Tab)
		{
			Binding: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "sort")),
			Action: func(m View) (tea.Cmd, bool) {
				// 1. 更新排序索引
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
				// A. 如果有批量选中，优先处理批量
				if len(v.selectedPids) > 0 {
					count := len(v.selectedPids)
					msg := fmt.Sprintf("Kill %d selected processes?", count)

					// 🔥 修复点：在 Action 里直接生成好 Batch 命令
					var cmds []tea.Cmd

					// 1. 遍历生成杀进程命令
					for pid := range v.selectedPids {
						cmds = append(cmds, v.killCmd(pid, false))
					}

					// 2. 追加一个“清空选中状态”的命令
					// 这样当 Batch 执行时，会发送这个消息给 Update
					cmds = append(cmds, func() tea.Msg { return ClearSelectionMsg{} })

					// 3. 把组合好的 Batch 命令传给弹窗
					// tea.Batch(...) 的返回值本身就是 tea.Cmd，完全匹配！
					return Push(NewConfirmDialog(msg, tea.Batch(cmds...))), true
				}

				// B. 如果没有选中，杀当前光标所在的单个进程
				if i := v.list.SelectedItem(); i != nil {
					var p core.Process
					if sp, ok := i.(SelectableProcess); ok {
						p = sp.Process
					} else if raw, ok := i.(core.Process); ok {
						p = raw
					} else {
						return nil, false
					}

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
		// 10. 空格键多选
		{
			Binding: key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "select")),
			Action: func(m View) (tea.Cmd, bool) {
				item := v.list.SelectedItem()
				if item == nil {
					return nil, false
				}

				// 获取 PID
				var pid int32
				if sp, ok := item.(SelectableProcess); ok {
					pid = sp.PID
				} else if p, ok := item.(core.Process); ok {
					pid = p.PID
				}

				// 切换选中状态
				if v.selectedPids[pid] {
					delete(v.selectedPids, pid)
				} else {
					v.selectedPids[pid] = true
				}

				// 立即刷新 UI (重新加上 [x])
				v.updateListItems()
				return nil, true
			},
		},
		// 11. 退出逻辑：优先清空选中，其次才是退出
		{
			Binding: key.NewBinding(key.WithKeys("esc", "q"), key.WithHelp("esc/q", "quit")),
			Action: func(m View) (tea.Cmd, bool) {
				// 1. 如果当前有选中的进程 -> 清空选中，退出多选模式
				if len(v.selectedPids) > 0 {
					v.selectedPids = make(map[int32]bool) // 清空 map
					v.updateListItems()                   // 刷新 UI (去掉 [x] 和缩进)
					return nil, true                      // 阻止后续退出逻辑
				}

				// 2. 如果当前是干净的 -> 弹出退出确认框
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
