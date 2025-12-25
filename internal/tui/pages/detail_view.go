package pages

import (
	"fmt"
	"strings"

	"github.com/Microindole/quell/internal/core"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ... 样式定义保持不变 ...
var (
	detailTitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FAFAFA")).Background(lipgloss.Color("#7D56F4")).Padding(0, 1).Bold(true)
	labelStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true).Width(10)
	detailBoxStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#7D56F4")).Padding(1, 2).MarginTop(1)
)

type DetailView struct {
	state    *SharedState
	registry *HandlerRegistry // 增加按键处理
	process  *core.Process
}

func NewDetailView(p *core.Process, state *SharedState) *DetailView {
	d := &DetailView{
		state:    state,
		registry: &HandlerRegistry{},
		process:  p,
	}
	d.registerActions()
	return d
}

func (d *DetailView) Init() tea.Cmd { return nil }

func (d *DetailView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {

	// 🔥 响应心跳：刷新当前进程数据
	case TickMsg:
		return d, d.refreshProcessCmd()

	// 🔥 接收刷新后的数据
	case *core.Process:
		d.process = msg
		return d, nil

	case tea.KeyMsg:
		if cmd, handled := d.registry.Handle(msg, d); handled {
			return d, cmd
		}
	}
	return d, nil
}

func (d *DetailView) registerActions() {
	// Back
	d.registry.Register(key.NewBinding(key.WithKeys("esc", "backspace"), key.WithHelp("esc", "back")),
		func(m View) (tea.Cmd, bool) {
			return Pop(), true
		})

	// Kill
	d.registry.Register(key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "kill")),
		func(m View) (tea.Cmd, bool) {
			// Push 弹窗，确认后 kill 并自动退回列表页 (Pop)
			cmd := tea.Batch(
				Pop(), // 关掉弹窗
				Pop(), // 关掉详情页(退回列表)
				func() tea.Msg { // 执行 Kill
					return processKilledMsg{err: d.state.Service.Kill(d.process.PID, false)}
				},
			)
			return Push(NewConfirmDialog(fmt.Sprintf("Kill %s?", d.process.Name), cmd)), true
		})
}

// 刷新单个进程数据
func (d *DetailView) refreshProcessCmd() tea.Cmd {
	return func() tea.Msg {
		// 简单起见，重新获取所有进程并找到当前这个
		// 这种做法虽然暴力但对本地进程监控来说性能足够，且能保证一致性
		procs, err := d.state.Service.GetProcesses()
		if err != nil {
			return nil
		}
		for _, p := range procs {
			if p.PID == d.process.PID {
				// 返回指针以避免大数据拷贝，需注意 core.Process 若由值传递改为指针更好
				// 这里假设 Process 是值类型，我们返回其指针给 Update
				newP := p
				return &newP
			}
		}
		// 如果找不到，说明进程已死
		return nil
	}
}

// View 和 ShortHelp 方法
func (d *DetailView) View() string {
	p := d.process
	memMB := float64(p.MemoryUsage) / 1024 / 1024

	// 格式化端口列表
	portStr := "None"
	if len(p.Ports) > 0 {
		var ps []string
		for _, port := range p.Ports {
			ps = append(ps, fmt.Sprintf("%d", port))
		}
		portStr = strings.Join(ps, ", ")
	}

	rows := []string{
		fmt.Sprintf("%s %s", labelStyle.Render("Name:"), p.Name),
		fmt.Sprintf("%s %d", labelStyle.Render("PID:"), p.PID),
		fmt.Sprintf("%s %s (%s)", labelStyle.Render("Port:"), portStr, p.Protocol),
		fmt.Sprintf("%s %s", labelStyle.Render("User:"), p.User),
		"",
		fmt.Sprintf("%s %.1f%%", labelStyle.Render("CPU:"), p.CpuPercent),
		fmt.Sprintf("%s %.1f MB", labelStyle.Render("Memory:"), memMB),
		"",
		labelStyle.Render("Command:"),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#A0A0A0")).Render(p.Cmdline),
	}
	return detailTitleStyle.Render(fmt.Sprintf(" Process Detail: %s ", p.Name)) + "\n" + detailBoxStyle.Render(strings.Join(rows, "\n"))
}

func (d *DetailView) ShortHelp() []key.Binding { return d.registry.MakeHelp() }
