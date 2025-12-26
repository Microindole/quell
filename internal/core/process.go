package core

import (
	"fmt"
	"strings"
)

// Connection 定义网络连接详情
type Connection struct {
	Fd         uint32
	Family     uint32
	Type       uint32
	LocalIP    string
	LocalPort  int
	RemoteIP   string
	RemotePort int
	Status     string // LISTEN, ESTABLISHED, CLOSE_WAIT...
}

type Process struct {
	PID        int32
	PPID       int32
	Name       string
	Ports      []int
	Protocol   string
	Status     string
	TreePrefix string

	Cmdline     string
	MemoryUsage uint64
	CpuPercent  float64
	User        string
	CreateTime  int64
}

func (p Process) FilterValue() string {
	var ports []string
	for _, port := range p.Ports {
		ports = append(ports, fmt.Sprintf(":%d", port))
	}
	portsStr := strings.Join(ports, " ")

	return fmt.Sprintf("%s %d %s %s", p.Name, p.PID, portsStr, p.Status)
}

func (p Process) IsSuspended() bool {
	s := strings.ToUpper(p.Status)
	return s == "T" || // Unix: Stopped
		s == "T+" || // Unix: Stopped (foreground)
		s == "SUSPENDED" || // Windows
		strings.Contains(s, "STOP") //包含 STOP 字样
}

func (p Process) Title() string {
	// 状态图标
	statusIcon := ""
	if p.IsSuspended() {
		statusIcon = "⏸️ "
	}

	// ---------------------------------------------------------
	// 🌳 模式 1: 树状视图 (Tree View)
	// ---------------------------------------------------------
	if p.TreePrefix != "" {
		memMB := float64(p.MemoryUsage) / 1024 / 1024
		nameDisplay := p.Name
		if p.IsSuspended() {
			nameDisplay += " [PAUSED]"
		}

		basic := fmt.Sprintf("%s%s%s", p.TreePrefix, statusIcon, nameDisplay)
		stats := fmt.Sprintf("  (PID:%d | %.1f%% | %.0fMB)", p.PID, p.CpuPercent, memMB)
		return basic + stats
	}

	// ---------------------------------------------------------
	// 📄 模式 2: 普通列表 (Flat View)
	// ---------------------------------------------------------

	// 端口显示优化
	portStr := ""
	if len(p.Ports) > 0 {
		if len(p.Ports) > 2 {
			portStr = fmt.Sprintf("(:%d...)", p.Ports[0])
		} else {
			var ps []string
			for _, port := range p.Ports {
				ps = append(ps, fmt.Sprintf(":%d", port))
			}
			portStr = fmt.Sprintf("(%s)", strings.Join(ps, ", "))
		}
	}

	displayName := p.Name
	if p.IsSuspended() {
		displayName = fmt.Sprintf("[PAUSED] %s", p.Name)
	}

	return fmt.Sprintf("%s%s %s", statusIcon, displayName, portStr)
}

func (p Process) Description() string {
	// 🌳 树状模式下隐藏
	if p.TreePrefix != "" {
		return ""
	}

	// 📄 普通模式：增加 Status 显示
	memMB := float64(p.MemoryUsage) / 1024 / 1024

	// 这里加了 Status 字段显示
	return fmt.Sprintf("PID: %d | CPU: %.1f%% | Mem: %.1f MB",
		p.PID, p.CpuPercent, memMB)
}

func (p Process) ShortCmd() string {
	if len(p.Cmdline) == 0 {
		return p.Name
	}
	if len(p.Cmdline) > 100 {
		return p.Cmdline[:97] + "..."
	}
	return strings.TrimSpace(p.Cmdline)
}
