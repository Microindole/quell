package pages

import (
	"fmt"
	"sort"
	"time"

	"github.com/Microindole/quell/internal/core"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type delayedRefreshMsg struct{}

var (
	// Logo 样式：使用更亮的紫色，增加一点 Margin
	logoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9F7AEA")). // 亮紫色
			Bold(true).
			MarginBottom(1)

	// 版本号样式 (Badge 风格)
	versionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1A1A1A")).
			Background(lipgloss.Color("#04B575")). // 绿色背景
			Padding(0, 1).
			Bold(true)

	// 加载文字样式
	loadingTextStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#626262")).
				Italic(true).
				MarginTop(1)

	quellLogo = `
   ____  __  __________    __ 
  / __ \/ / / / ____/ /   / / 
 / / / / / / / __/ / /   / /  
/ /_/ / /_/ / /___/ /___/ /___
\___\_\____/_____/_____/_____/
`
)

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
		v.list.SetSize(msg.Width-4, msg.Height-4)

	case TickMsg:
		return v, v.refreshListCmd()

	case []list.Item:
		v.loading = false // 加载完成，Loading 界面消失

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
			v.status = fmt.Sprintf("Scanned %d processes.", len(msg))
		}

		cmd = v.list.SetItems(finalItems)
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
		w, h := v.list.Width(), v.list.Height()
		if w == 0 || h == 0 {
			w, h = 80, 24
		}

		// 组合内容：
		// Logo
		// Version Badge (v1.0.0)
		// Loading Text
		content := lipgloss.JoinVertical(lipgloss.Center,
			logoStyle.Render(quellLogo),
			versionStyle.Render(" v1.0.0 "),                        // 这里写死或从 config 读
			loadingTextStyle.Render("Initializing Neural Link..."), // 搞点中二的提示语
		)

		return lipgloss.Place(
			w, h,
			lipgloss.Center, lipgloss.Center,
			content,
		)
	}
	return v.list.View()
}

func (v *ListView) ShortHelp() []key.Binding { return v.registry.MakeHelp() }

func (v *ListView) registerActions() {
	actions := GetDefaultListActions(v)
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
