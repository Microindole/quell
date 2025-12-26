package pages

import (
	"fmt"
	"strings"

	"github.com/Microindole/quell/internal/core"
	"github.com/Microindole/quell/internal/tui/components" // 引入组件包
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	detailTitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FAFAFA")).Background(lipgloss.Color("#7D56F4")).Padding(0, 1).Bold(true)
	labelStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true).Width(10)
	detailBoxStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#7D56F4")).Padding(1, 2).MarginTop(1)
	cpuColor         = lipgloss.Color("#04B575")
	memColor         = lipgloss.Color("#7D56F4")
	connHeaderStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FAFAFA")).Background(lipgloss.Color("#626262")).Padding(0, 1)
	connRowStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#A0A0A0"))
)

const maxHistory = 40

type ProcessConnectionsMsg []core.Connection

type DetailView struct {
	state       *SharedState
	registry    *HandlerRegistry
	process     *core.Process
	cpuHistory  []float64
	memHistory  []float64
	width       int
	cpuChart    *components.Sparkline
	memChart    *components.Sparkline
	connections []core.Connection
}

func NewDetailView(p *core.Process, state *SharedState, width int) *DetailView {
	d := &DetailView{
		state:       state,
		registry:    &HandlerRegistry{},
		process:     p,
		cpuHistory:  make([]float64, maxHistory),
		memHistory:  make([]float64, maxHistory),
		width:       width,
		cpuChart:    components.NewSparkline(lipgloss.NewStyle().Foreground(cpuColor)),
		memChart:    components.NewSparkline(lipgloss.NewStyle().Foreground(memColor)),
		connections: nil,
	}
	d.registerActions()
	return d
}

func (d *DetailView) Init() tea.Cmd {
	return d.fetchConnectionsCmd()
}

func (d *DetailView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		d.width = msg.Width
		return d, nil

	case TickMsg:
		return d, d.refreshProcessCmd()

	case *core.Process:
		d.process = msg
		// 更新数据历史
		d.cpuHistory = d.cpuHistory[1:]
		d.cpuHistory = append(d.cpuHistory, msg.CpuPercent)

		memMB := float64(msg.MemoryUsage) / 1024 / 1024
		d.memHistory = d.memHistory[1:]
		d.memHistory = append(d.memHistory, memMB)

		return d, nil

	case ProcessConnectionsMsg:
		d.connections = msg
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

func (d *DetailView) refreshProcessCmd() tea.Cmd {
	return func() tea.Msg {
		procs, err := d.state.Service.GetProcesses()
		if err != nil {
			return nil
		}
		for _, p := range procs {
			if p.PID == d.process.PID {
				newP := p
				return &newP
			}
		}
		return nil
	}
}

func (d *DetailView) fetchConnectionsCmd() tea.Cmd {
	return func() tea.Msg {
		conns, err := d.state.Service.GetConnections(d.process.PID)
		if err != nil {
			return nil
		}
		return ProcessConnectionsMsg(conns)
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

	cpuGraph := d.cpuChart.Render(d.cpuHistory)
	memGraph := d.memChart.Render(d.memHistory)

	maxWidth := d.width - 12
	if maxWidth < 20 {
		maxWidth = 20
	}

	cpuVal := fmt.Sprintf("%.1f%%", p.CpuPercent)
	memVal := fmt.Sprintf("%.1f MB", memMB)

	var connSection string
	if len(d.connections) > 0 {
		// 有数据：显示表头和前几条
		lines := []string{connHeaderStyle.Render(fmt.Sprintf("%-6s | %-21s | %-21s | %s", "Proto", "Local", "Remote", "Status"))}

		limit := 5 // 只显示前 5 条
		for i, c := range d.connections {
			if i >= limit {
				lines = append(lines, connRowStyle.Render(fmt.Sprintf("... and %d more", len(d.connections)-limit)))
				break
			}
			// 处理 0.0.0.0
			remote := fmt.Sprintf("%s:%d", c.RemoteIP, c.RemotePort)
			if c.RemotePort == 0 {
				remote = "*"
			}

			row := fmt.Sprintf("%-6s | %-21s | %-21s | %s", "TCP",
				fmt.Sprintf("%s:%d", c.LocalIP, c.LocalPort),
				remote,
				c.Status,
			)
			lines = append(lines, connRowStyle.Render(row))
		}
		connSection = "\n\n" + strings.Join(lines, "\n")
	} else {
		// 无数据：提示可能是权限问题
		connSection = "\n\n" + connRowStyle.Render("(No connections or permission denied. Try sudo?)")
	}

	cmdDisplay := p.Cmdline
	if len(cmdDisplay) > maxWidth {
		cmdDisplay = cmdDisplay[:maxWidth-3] + "..."
	}

	cmdStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#A0A0A0")).
		Width(maxWidth).
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
		cmdStyle.Render(cmdDisplay), // 使用截断后的字符串
		labelStyle.Render("Network:"),
		connSection,
	}
	return detailTitleStyle.Render(fmt.Sprintf(" Process Detail: %s ", p.Name)) + "\n" + detailBoxStyle.Render(strings.Join(rows, "\n"))
}

func (d *DetailView) ShortHelp() []key.Binding { return d.registry.MakeHelp() }
