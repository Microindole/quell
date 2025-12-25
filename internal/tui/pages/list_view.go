package pages

import (
	"fmt"
	"sort"
	"time"

	"github.com/Microindole/quell/internal/core"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type delayedRefreshMsg struct{}

type ListView struct {
	state          *SharedState
	list           list.Model
	registry       *HandlerRegistry
	sorters        []Sorter
	currentSortIdx int
	loading        bool
	status         string
	treeMode       bool
}

func NewListView(state *SharedState, sortIdx int, treeMode bool) *ListView {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Quell - Process Killer"
	l.SetShowHelp(false)

	v := &ListView{
		state:          state,
		list:           l,
		registry:       &HandlerRegistry{},
		sorters:        []Sorter{StatusSorter{}, CPUSorter{}, MemSorter{}, PIDSorter{}},
		currentSortIdx: sortIdx,
		treeMode:       treeMode,
		loading:        true,
		status:         "Scanning...",
	}
	if treeMode {
		v.status = "Wait for scan (Tree View)..."
	}
	v.registerActions()
	return v
}

// GetState 导出当前状态
func (v *ListView) GetState() (int, bool) {
	return v.currentSortIdx, v.treeMode
}

func (v *ListView) Init() tea.Cmd {
	// 🔥 Init 不再启动 Tick，只启动数据刷新
	return v.refreshListCmd()
}

func (v *ListView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.list.SetSize(msg.Width-4, msg.Height-4)

	// 🔥 收到 TickMsg，只负责刷新数据，不要再发 TickCmd 了 (Model 已经发了)
	case TickMsg:
		return v, v.refreshListCmd()

	case []list.Item:
		v.loading = false

		delegate := list.NewDefaultDelegate()
		if v.treeMode {
			delegate.ShowDescription = false
			delegate.SetSpacing(0)
		} else {
			delegate.ShowDescription = true
			delegate.SetSpacing(0)
		}
		v.list.SetDelegate(delegate)

		var rawProcs []core.Process
		for _, item := range msg {
			rawProcs = append(rawProcs, item.(core.Process))
		}
		var finalItems []list.Item
		if v.treeMode {
			treeProcs := BuildTree(rawProcs)
			finalItems = make([]list.Item, len(treeProcs))
			for i, p := range treeProcs {
				finalItems[i] = p
			}
			// 🔥 回归简单：直接更新状态，不判断前缀
			v.status = fmt.Sprintf("Tree View: %d procs", len(msg))
		} else {
			for i := range rawProcs {
				rawProcs[i].TreePrefix = ""
			}
			items := make([]list.Item, len(rawProcs))
			for i, p := range rawProcs {
				items[i] = p
			}
			finalItems = v.sortItems(items)

			// 🔥 回归简单：直接更新状态
			v.status = fmt.Sprintf("Scanned %d processes.", len(msg))
		}

		cmd = v.list.SetItems(finalItems)
		return v, cmd

	case ProcessActionMsg:
		if msg.Err != nil {
			v.status = fmt.Sprintf("Error: %v", msg.Err)
			return v, nil
		}
		// 动态显示操作结果：Killed, Suspended, Resumed
		v.status = fmt.Sprintf("%s successfully.", msg.Action)
		return v, v.delayedRefreshCmd()

	case delayedRefreshMsg:
		return v, v.refreshListCmd()

	case tea.KeyMsg:
		if v.list.FilterState() == list.Filtering {
			v.list, cmd = v.list.Update(msg)
			return v, cmd
		}
		if cmd, handled := v.registry.Handle(msg, v); handled {
			return v, cmd
		}
	}

	v.list, cmd = v.list.Update(msg)
	cmds = append(cmds, cmd)
	return v, tea.Batch(cmds...)
}

func (v *ListView) View() string {
	if v.loading {
		return "Loading..."
	}
	return v.list.View()
}
func (v *ListView) ShortHelp() []key.Binding { return v.registry.MakeHelp() }

func (v *ListView) registerActions() {
	// 🔥 使用命令模式重构：获取所有定义好的命令
	actions := GetDefaultListActions(v)

	// 统一注册
	for _, action := range actions {
		v.registry.Register(action.Binding, action.Action)
	}
}

func (v *ListView) sortItems(items []list.Item) []list.Item {
	sorted := make([]list.Item, len(items))
	copy(sorted, items)
	sorter := v.sorters[v.currentSortIdx]
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorter.Less(sorted[i].(core.Process), sorted[j].(core.Process))
	})
	return sorted
}

func (v *ListView) refreshListCmd() tea.Cmd {
	return func() tea.Msg {
		procs, err := v.state.Service.GetProcesses()
		if err != nil {
			return nil
		}
		items := make([]list.Item, len(procs))
		for i, p := range procs {
			items[i] = p
		}
		return items
	}
}
func (v *ListView) killCmd(pid int32, force bool) tea.Cmd {
	return func() tea.Msg {
		return ProcessActionMsg{
			Err:    v.state.Service.Kill(pid, force),
			Action: "Killed",
		}
	}
}
func (v *ListView) delayedRefreshCmd() tea.Cmd {
	return tea.Tick(1, func(t time.Time) tea.Msg { return delayedRefreshMsg{} })
}

func (v *ListView) GetStatus() string { return v.status }

func (v *ListView) GetSortName() string { return v.sorters[v.currentSortIdx].Name() }
