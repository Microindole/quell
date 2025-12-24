package core

import (
	"fmt"
	"strings"
)

type Process struct {
	PID      int32
	Name     string
	Port     int    // 监听端口
	Protocol string // TCP/UDP

	Cmdline     string  // 完整的启动命令 (e.g., "/usr/bin/node server.js")
	MemoryUsage uint64  // 内存占用 (字节 Byte)
	CpuPercent  float64 // CPU 使用率 (百分比)
	User        string  // 启动该进程的用户
}

func (p Process) FilterValue() string {
	// 允许用户搜名字，也允许搜 PID 或端口
	return fmt.Sprintf("%s %d %d", p.Name, p.PID, p.Port)
}

func (p Process) Title() string {
	return fmt.Sprintf("%s (:%d)", p.Name, p.Port)
}

func (p Process) Description() string {
	memMB := float64(p.MemoryUsage) / 1024 / 1024
	return fmt.Sprintf("PID: %d | CPU: %.1f%% | Mem: %.1f MB", p.PID, p.CpuPercent, memMB)
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
