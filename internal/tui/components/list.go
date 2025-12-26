package components

import (
	"github.com/Microindole/quell/internal/core"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ProcessItem 定义列表项接口，解耦具体数据
type ProcessItem interface {
	list.Item
	GetProcess() core.Process
	IsSelected() bool
}

// ConcreteItem 实现 ProcessItem，用于组件内部传输
type ConcreteItem struct {
	Process      core.Process
	Selected     bool
	ShowCheckbox bool
}

func (i ConcreteItem) Title() string {
	prefix := ""
	// 如果需要显示 Checkbox（多选模式）
	if i.ShowCheckbox {
		prefix = "[ ] "
		if i.Selected {
			prefix = "[x] "
		}
	}
	// 调用 core.Process 自身的 Title 逻辑 (包含 TreePrefix 处理)
	return prefix + i.Process.Title()
}

func (i ConcreteItem) Description() string      { return i.Process.Description() }
func (i ConcreteItem) FilterValue() string      { return i.Process.FilterValue() }
func (i ConcreteItem) GetProcess() core.Process { return i.Process }
func (i ConcreteItem) IsSelected() bool         { return i.Selected }

// ProcessList 封装 list.Model
type ProcessList struct {
	Model    list.Model
	delegate list.DefaultDelegate
}

func NewProcessList(width, height int) *ProcessList {
	d := list.NewDefaultDelegate()

	// 使用更柔和的高亮效果
	d.Styles.FilterMatch = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7D56F4")). // 紫色高亮
		Bold(true)

	d.ShowDescription = true
	d.SetSpacing(0)

	l := list.New([]list.Item{}, d, width, height)
	l.Title = "Quell - Process Killer"
	l.SetShowHelp(false)

	return &ProcessList{
		Model:    l,
		delegate: d,
	}
}

func (p *ProcessList) Init() tea.Cmd {
	return nil
}

func (p *ProcessList) Update(msg tea.Msg) (*ProcessList, tea.Cmd) {
	var cmd tea.Cmd
	p.Model, cmd = p.Model.Update(msg)
	return p, cmd
}

func (p *ProcessList) View() string {
	return p.Model.View()
}

// SetItems 封装数据转换逻辑：外部只传 core.Process，组件自己封装成 ListItem
func (p *ProcessList) SetItems(procs []core.Process, selectedPids map[int32]bool) tea.Cmd {
	items := make([]list.Item, len(procs))
	hasSelection := len(selectedPids) > 0

	for i, proc := range procs {
		items[i] = ConcreteItem{
			Process:      proc,
			Selected:     selectedPids[proc.PID],
			ShowCheckbox: hasSelection,
		}
	}
	return p.Model.SetItems(items)
}

// SelectedItem 安全获取当前选中的进程
func (p *ProcessList) SelectedItem() *core.Process {
	if i := p.Model.SelectedItem(); i != nil {
		if pi, ok := i.(ProcessItem); ok {
			val := pi.GetProcess()
			return &val
		}
	}
	return nil
}

func (p *ProcessList) SetSize(w, h int) {
	p.Model.SetSize(w, h)
}

// Inner 暴露底层 Model，仅用于需要访问 FilterState 等特殊场景
func (p *ProcessList) Inner() *list.Model {
	return &p.Model
}

func (p *ProcessList) SetTreeMode(isTree bool) {
	if isTree {
		p.delegate.ShowDescription = false
	} else {
		p.delegate.ShowDescription = true
	}
	// 重新设置 delegate
	p.Model.SetDelegate(p.delegate)
}
