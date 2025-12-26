package pages

import (
	"fmt"
	"sort"
	"time"

	"github.com/Microindole/quell/internal/core"
	"github.com/Microindole/quell/internal/tui/components" // 引用组件
	"github.com/Microindole/quell/internal/version"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type delayedRefreshMsg struct{}

var (
	logoStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#9F7AEA")).Bold(true).MarginBottom(1)
	versionStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#1A1A1A")).Background(lipgloss.Color("#04B575")).Padding(0, 1).Bold(true)
	loadingTextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Italic(true).MarginTop(1)
	quellLogo        = `
   ____  __  __________    __ 
  / __ \/ / / / ____/ /   / / 
 / / / / / / / __/ / /   / /  
/ /_/ / /_/ / /___/ /___/ /___
\___\_\____/_____/_____/_____/
`
)

type ClearSelectionMsg struct{}

// ListView 现在是 Controller 角色
type ListView struct {
	state *SharedState

	// 🔥 核心变化：使用封装后的组件
	processList *components.ProcessList

	registry       *HandlerRegistry
	sorters        []Sorter
	currentSortIdx int
	loading        bool
	status         string
	treeMode       bool
	selectedPids   map[int32]bool
	rawProcesses   []core.Process
}

func NewListView(state *SharedState, sortIdx int, treeMode bool) *ListView {
	// 初始化组件
	pl := components.NewProcessList(0, 0)

	v := &ListView{
		state:          state,
		processList:    pl,
		registry:       &HandlerRegistry{},
		sorters:        []Sorter{StatusSorter{}, CPUSorter{}, MemSorter{}, PIDSorter{}},
		currentSortIdx: sortIdx,
		treeMode:       treeMode,
		loading:        true,
		status:         "Scanning...",
		selectedPids:   make(map[int32]bool),
	}
	if treeMode {
		v.status = "Wait for scan (Tree View)..."
	}
	v.registerActions()
	return v
}

func (v *ListView) GetState() (int, bool) {
	return v.currentSortIdx, v.treeMode
}

func (v *ListView) Init() tea.Cmd {
	return v.refreshListCmd()
}

func (v *ListView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.processList.SetSize(msg.Width-4, msg.Height-4)

	case ClearSelectionMsg:
		v.selectedPids = make(map[int32]bool)
		cmd = v.updateListItems()
		cmds = append(cmds, cmd)
		return v, tea.Batch(cmds...)

	case TickMsg:
		return v, v.refreshListCmd()

	case []list.Item:
		var rawProcs []core.Process
		for _, item := range msg {
			if p, ok := item.(core.Process); ok {
				rawProcs = append(rawProcs, p)
			}
		}
		v.loading = false
		v.rawProcesses = rawProcs
		cmd = v.updateListItems()
		return v, cmd

	case ProcessActionMsg:
		if msg.Err != nil {
			v.status = fmt.Sprintf("Error: %v", msg.Err)
			return v, nil
		}
		v.status = fmt.Sprintf("%s successfully.", msg.Action)
		return v, v.delayedRefreshCmd()

	case delayedRefreshMsg:
		return v, v.refreshListCmd()

	case tea.KeyMsg:
		isSearching := v.processList.Inner().FilterState() == list.Filtering || v.processList.Inner().FilterInput.Value() != ""

		if isSearching {
			if msg.String() == "esc" {
				v.processList.Inner().ResetFilter()
				return v, nil
			}
			if msg.String() == " " {
				if p := v.processList.SelectedItem(); p != nil {
					if v.selectedPids[p.PID] {
						delete(v.selectedPids, p.PID)
					} else {
						v.selectedPids[p.PID] = true
					}
					return v, v.updateListItems()
				}
				return v, nil
			}

			v.processList, cmd = v.processList.Update(msg)
			return v, cmd
		}
		if cmd, handled := v.registry.Handle(msg, v); handled {
			return v, cmd
		}
	case SetFilterMsg:
		// 1. 设置输入框的值
		v.processList.Inner().FilterInput.SetValue(string(msg))
		v.processList.Inner().SetFilterState(list.Filtering)
		return v, v.updateListItems()
	}

	v.processList, cmd = v.processList.Update(msg)
	cmds = append(cmds, cmd)
	return v, tea.Batch(cmds...)
}

func (v *ListView) updateListItems() tea.Cmd {
	// 调整组件样式
	v.processList.SetTreeMode(v.treeMode)

	filterVal := v.processList.Inner().FilterInput.Value()
	currentFilterState := v.processList.Inner().FilterState()

	var finalProcs []core.Process

	// 准备数据
	if v.treeMode {
		treeProcs := BuildTree(v.rawProcesses)
		finalProcs = treeProcs
		if len(v.selectedPids) > 0 {
			v.status = fmt.Sprintf("%d selected | Tree View", len(v.selectedPids))
		} else {
			v.status = fmt.Sprintf("Tree View: %d procs", len(v.rawProcesses))
		}
	} else {
		sortedRaw := make([]core.Process, len(v.rawProcesses))
		copy(sortedRaw, v.rawProcesses)
		sorter := v.sorters[v.currentSortIdx]
		sort.SliceStable(sortedRaw, func(i, j int) bool {
			return sorter.Less(sortedRaw[i], sortedRaw[j])
		})

		// 清除 TreePrefix
		for i := range sortedRaw {
			sortedRaw[i].TreePrefix = ""
		}

		finalProcs = sortedRaw
		if len(v.selectedPids) > 0 {
			v.status = fmt.Sprintf("%d selected | Total: %d", len(v.selectedPids), len(v.rawProcesses))
		} else {
			v.status = fmt.Sprintf("Scanned %d processes.", len(v.rawProcesses))
		}
	}

	cmd := v.processList.SetItems(finalProcs, v.selectedPids)

	if filterVal != "" {
		v.processList.Inner().FilterInput.SetValue(filterVal)
	}
	v.processList.Inner().SetFilterState(currentFilterState)

	return cmd
}

func (v *ListView) View() string {
	if v.loading {
		w, h := v.processList.Inner().Width(), v.processList.Inner().Height()
		if w == 0 || h == 0 {
			w, h = 80, 24
		}
		content := lipgloss.JoinVertical(lipgloss.Center,
			logoStyle.Render(quellLogo),
			versionStyle.Render(" "+version.Version+" "),
			loadingTextStyle.Render("Initializing Neural Link..."),
		)
		return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, content)
	}
	// 🔥 使用组件渲染
	return v.processList.View()
}

func (v *ListView) ShortHelp() []key.Binding { return v.registry.MakeHelp() }

func (v *ListView) registerActions() {
	actions := GetDefaultListActions(v)
	for _, action := range actions {
		v.registry.Register(action.Binding, action.Action)
	}
}

func (v *ListView) refreshListCmd() tea.Cmd {
	return func() tea.Msg {
		procs, err := v.state.Service.GetProcesses()
		if err != nil {
			return nil
		}
		// 为了保持 Init 接口兼容，这里我们先转成 []list.Item
		// (或者你可以直接改 Service 返回类型处理，但为了最小改动，这里做个转换层)
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

func (v *ListView) GetStatus() string   { return v.status }
func (v *ListView) GetSortName() string { return v.sorters[v.currentSortIdx].Name() }
