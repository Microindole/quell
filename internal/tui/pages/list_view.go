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

// SelectableProcess
// 它的作用是仅仅在 UI 层重写 Title，加上选中标记
type SelectableProcess struct {
	core.Process
	Selected     bool
	ShowCheckbox bool
}

func (s SelectableProcess) Title() string {
	// 如果不在多选模式，直接返回原始标题 (干干净净，没有 [ ])
	if !s.ShowCheckbox {
		return s.Process.Title()
	}

	// 如果在多选模式，显示 [x] 或 [ ]
	prefix := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render("[ ] ")
	if s.Selected {
		prefix = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Render("[x] ")
	}
	return prefix + s.Process.Title()
}

func (s SelectableProcess) FilterValue() string {
	return s.Process.FilterValue()
}

type ClearSelectionMsg struct{}

type ListView struct {
	state          *SharedState
	list           list.Model
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
		v.list.SetSize(msg.Width-4, msg.Height-4)

	case ClearSelectionMsg:
		v.selectedPids = make(map[int32]bool) // 清空 map
		v.updateListItems()                   // 强制刷新列表 UI (去掉 [x])
		return v, nil

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
		v.rawProcesses = rawProcs // 缓存原始数据

		// 🔥 调用统一的更新列表方法
		v.updateListItems()
		return v, nil

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

func (v *ListView) updateListItems() {
	delegate := list.NewDefaultDelegate()
	if v.treeMode {
		delegate.ShowDescription = false
		delegate.SetSpacing(0)
	} else {
		delegate.ShowDescription = true
		delegate.SetSpacing(0)
	}
	v.list.SetDelegate(delegate)

	hasSelection := len(v.selectedPids) > 0

	var finalItems []list.Item

	if v.treeMode {
		treeProcs := BuildTree(v.rawProcesses)
		finalItems = make([]list.Item, len(treeProcs))
		for i, p := range treeProcs {
			finalItems[i] = SelectableProcess{
				Process:      p,
				Selected:     v.selectedPids[p.PID],
				ShowCheckbox: hasSelection,
			}
		}

		// 状态栏文案
		if hasSelection {
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

		finalItems = make([]list.Item, len(sortedRaw))
		for i, p := range sortedRaw {
			p.TreePrefix = ""
			finalItems[i] = SelectableProcess{
				Process:      p,
				Selected:     v.selectedPids[p.PID],
				ShowCheckbox: hasSelection,
			}
		}
		// 状态栏文案
		if hasSelection {
			v.status = fmt.Sprintf("%d selected | Total: %d", len(v.selectedPids), len(v.rawProcesses))
		} else {
			v.status = fmt.Sprintf("Scanned %d processes.", len(v.rawProcesses))
		}
	}

	v.list.SetItems(finalItems)
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
