package core

import (
	"fmt"
	"strings"
)

type Process struct {
	PID        int32
	PPID       int32
	Name       string
	Ports      []int
	Protocol   string
	Status     string
	TreePrefix string // 树状图前缀

	Cmdline     string
	MemoryUsage uint64
	CpuPercent  float64
	User        string
}

func (p Process) FilterValue() string {
	portsStr := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(p.Ports)), " "), "[]")
	return fmt.Sprintf("%s %d %s %s", p.Name, p.PID, portsStr, p.Status)
}

func (p Process) Title() string {
	// 状态图标
	statusIcon := ""
	if p.Status == "T" {
		statusIcon = "⏸️ "
	}

	// ---------------------------------------------------------
	// 🌳 模式 1: 树状视图 (Tree View)
	// ---------------------------------------------------------
	if p.TreePrefix != "" {
		// 单行显示：[前缀] [图标] [名字] ... [统计数据]

		memMB := float64(p.MemoryUsage) / 1024 / 1024

		displayIcon := statusIcon
		if displayIcon == "" {
			displayIcon = " "
		}

		basic := fmt.Sprintf("%s%s%s", p.TreePrefix, displayIcon, p.Name)

		// 统计部分：跟在后面
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
	return fmt.Sprintf("%s%s %s", statusIcon, p.Name, portStr)
}

func (p Process) Description() string {
	// 🌳 树状模式下，必须隐藏第二行，否则竖线会断开！
	if p.TreePrefix != "" {
		return ""
	}

	// 📄 普通模式：显示详情
	memMB := float64(p.MemoryUsage) / 1024 / 1024
	return fmt.Sprintf("PID: %d | PPID: %d | CPU: %.1f%% | Mem: %.1f MB", p.PID, p.PPID, p.CpuPercent, memMB)
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
