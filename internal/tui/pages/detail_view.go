package pages

import (
	"fmt"
	"strings"

	"github.com/Microindole/quell/internal/core"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	detailTitleStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FAFAFA")).Background(lipgloss.Color("#7D56F4")).Padding(0, 1).Bold(true)
	labelStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true).Width(10)
	detailBoxStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#7D56F4")).Padding(1, 2).MarginTop(1)
	cpuSparklineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	memSparklineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))
)

const maxHistory = 40

type DetailView struct {
	state      *SharedState
	registry   *HandlerRegistry // 增加按键处理
	process    *core.Process
	cpuHistory []float64
	memHistory []float64
	width      int
}

func NewDetailView(p *core.Process, state *SharedState, width int) *DetailView {
	d := &DetailView{
		state:      state,
		registry:   &HandlerRegistry{},
		process:    p,
		cpuHistory: make([]float64, maxHistory),
		memHistory: make([]float64, maxHistory),
		width:      width, // 初始化宽度
	}
	d.registerActions()
	return d
}

func (d *DetailView) Init() tea.Cmd { return nil }

func (d *DetailView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		d.width = msg.Width
		return d, nil

	case TickMsg:
		return d, d.refreshProcessCmd()

	case *core.Process:
		d.process = msg

		// 1. 更新 CPU 历史
		d.cpuHistory = d.cpuHistory[1:]
		d.cpuHistory = append(d.cpuHistory, msg.CpuPercent)

		// 2. 更新 Memory 历史 (单位转为 MB，保持数据量级一致)
		memMB := float64(msg.MemoryUsage) / 1024 / 1024
		d.memHistory = d.memHistory[1:]
		d.memHistory = append(d.memHistory, memMB)

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
			cmd := tea.Batch(
				Pop(),
				Pop(),
				func() tea.Msg {
					return ProcessActionMsg{
						Err:    d.state.Service.Kill(d.process.PID, false),
						Action: "Killed",
					}
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

func (d *DetailView) View() string {
	p := d.process
	memMB := float64(p.MemoryUsage) / 1024 / 1024

	portStr := "None"
	if len(p.Ports) > 0 {
		var ps []string
		for _, port := range p.Ports {
			ps = append(ps, fmt.Sprintf("%d", port))
		}
		portStr = strings.Join(ps, ", ")
	}

	cpuGraph := cpuSparklineStyle.Render(renderSparkline(d.cpuHistory))
	memGraph := memSparklineStyle.Render(renderSparkline(d.memHistory))

	maxWidth := d.width - 12
	if maxWidth < 20 {
		maxWidth = 20 // 最小保护
	}

	cpuVal := fmt.Sprintf("%.1f%%", p.CpuPercent)
	memVal := fmt.Sprintf("%.1f MB", memMB)

	// 定义 Command 样式，强制换行
	cmdStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#A0A0A0")).
		Width(maxWidth). // 关键：设置宽度触发自动换行
		Align(lipgloss.Left)

	rows := []string{
		fmt.Sprintf("%s %s", labelStyle.Render("Name:"), p.Name),
		fmt.Sprintf("%s %d", labelStyle.Render("PID:"), p.PID),
		fmt.Sprintf("%s %s (%s)", labelStyle.Render("Port:"), portStr, p.Protocol),
		fmt.Sprintf("%s %s", labelStyle.Render("User:"), p.User),
		"",
		fmt.Sprintf("%s %-12s %s", labelStyle.Render("CPU:"), cpuVal, cpuGraph),
		fmt.Sprintf("%s %-12s %s", labelStyle.Render("Memory:"), memVal, memGraph),
		"",
		labelStyle.Render("Command:"),
		cmdStyle.Render(p.Cmdline),
	}
	return detailTitleStyle.Render(fmt.Sprintf(" Process Detail: %s ", p.Name)) + "\n" + detailBoxStyle.Render(strings.Join(rows, "\n"))
}

func (d *DetailView) ShortHelp() []key.Binding { return d.registry.MakeHelp() }

// renderSparkline 将浮点数切片转换为方块字符图
// renderSparkline 将浮点数切片转换为波形图
func renderSparkline(data []float64) string {
	if len(data) == 0 {
		return ""
	}

	m := 0.0
	for _, v := range data {
		if v > m {
			m = v
		}
	}

	// 动态调整基准：
	// 如果最大值很小（比如内存波动只有 0.1MB），我们设置一个最小基准，避免噪点被放大成巨浪。
	// 对于 CPU，满载是 100，但为了看清微小波动，我们可以设低一点的 floor。
	if m < 1.0 {
		m = 1.0
	}

	// 优化字符集：移除空格，使用 " ▂▃▄▅▆▇█"
	// 第一个字符是 U+2581 (Lower One Eighth Block)，保证有基准线
	levels := []rune(" ▂▃▄▅▆▇█")

	var sb strings.Builder
	for _, v := range data {
		// 计算高度比例 (0.0 - 1.0)
		ratio := v / m

		// 映射到索引 (0 - 7)
		idx := int(ratio * float64(len(levels)-1))

		// 边界保护
		if idx < 0 {
			idx = 0
		}
		if idx >= len(levels) {
			idx = len(levels) - 1
		}

		sb.WriteRune(levels[idx])
	}
	return sb.String()
}
