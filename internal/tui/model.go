package tui

import (
	"sort"
	"time"

	"github.com/Microindole/quell/internal/core"
	"github.com/Microindole/quell/internal/system"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

const heartbeatInterval = 2 * time.Second

type tickMsg time.Time

type delayedRefreshMsg struct{}
type processKilledMsg struct{ err error }

type Model struct {
	list           list.Model
	svc            *core.Service
	registry       *HandlerRegistry
	sorters        []Sorter
	currentSortIdx int
	loading        bool
	status         string
	inspecting     bool
	selected       *core.Process
	isAdmin        bool
}

func NewModel(svc *core.Service) Model {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Quell - Process Killer"
	l.SetShowHelp(false)

	m := Model{
		list:     l,
		svc:      svc,
		registry: &HandlerRegistry{},
		sorters: []Sorter{
			PIDSorter{},
			MemSorter{},
			CPUSorter{},
		},
		currentSortIdx: 0,
		loading:        true,
		status:         "Scanning ports...",
		isAdmin:        system.IsAdmin(),
	}
	registerCoreActions(&m)
	registerSortActions(&m)

	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.refreshListCmd(),
		m.tickCmd(),
	)
}

func (m Model) sortItems(items []list.Item) []list.Item {
	sorted := make([]list.Item, len(items))
	copy(sorted, items)

	currentSorter := m.sorters[m.currentSortIdx]

	sort.SliceStable(sorted, func(i, j int) bool {
		p1 := sorted[i].(core.Process)
		p2 := sorted[j].(core.Process)
		return currentSorter.Less(p1, p2)
	})
	return sorted
}

func (m Model) refreshListCmd() tea.Cmd {
	return func() tea.Msg {
		procs, err := m.svc.GetProcesses()
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

func (m Model) killProcessCmd(pid int32, force bool) tea.Cmd {
	return func() tea.Msg {
		err := m.svc.Kill(pid, force)
		return processKilledMsg{err: err}
	}
}

func (m Model) delayedRefreshCmd() tea.Cmd {
	return tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
		return delayedRefreshMsg{}
	})
}

func (m Model) getTarget() (int32, string) {
	if m.inspecting && m.selected != nil {
		return m.selected.PID, m.selected.Name
	}
	if i := m.list.SelectedItem(); i != nil {
		p := i.(core.Process) // 类型断言
		return p.PID, p.Name
	}
	return 0, ""
}

func (m Model) tickCmd() tea.Cmd {
	return tea.Tick(heartbeatInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) getSortName() string {
	if len(m.sorters) == 0 {
		return "Unknown"
	}
	return m.sorters[m.currentSortIdx].Name()
}
