package system

import (
	"path/filepath"
	"strings"

	"github.com/Microindole/quell/internal/core"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// LocalProvider 实现了 core.Provider 接口
type LocalProvider struct{}

func NewLocalProvider() *LocalProvider {
	return &LocalProvider{}
}

// ListProcesses 扫描本地 TCP 监听端口
func (l *LocalProvider) ListProcesses() ([]core.Process, error) {
	conns, err := net.Connections("tcp")
	if err != nil {
		return nil, err
	}

	var results []core.Process

	for _, conn := range conns {
		if conn.Status != "LISTEN" {
			continue
		}
		if conn.Pid == 0 {
			continue
		}

		// 1. 基础信息
		pid := conn.Pid
		port := int(conn.Laddr.Port)

		// 2. 获取 Process 对象
		p, err := process.NewProcess(pid)
		if err != nil {
			// 进程可能刚消失
			continue
		}

		// 3. 填充详细信息 (Phase 1 新增)
		// 注意：这些操作可能会因为权限问题失败，我们尽量获取，失败就给默认值
		name := l.getName(p)

		cmdline, _ := p.Cmdline()

		// 获取内存信息 (RSS)
		memInfo, _ := p.MemoryInfo()
		var memUsage uint64
		if memInfo != nil {
			memUsage = memInfo.RSS
		}

		// 获取 CPU (注意：Percent(0) 表示计算从上次调用以来的间隔，第一次调用可能不准，但在列表中还行)
		cpuPercent, _ := p.Percent(0)

		// 获取用户名
		user, _ := p.Username()

		results = append(results, core.Process{
			PID:         pid,
			Name:        name,
			Port:        port,
			Protocol:    "TCP",
			Cmdline:     cmdline,
			MemoryUsage: memUsage,
			CpuPercent:  cpuPercent,
			User:        user,
		})
	}

	return results, nil
}

// Kill 杀进程
func (l *LocalProvider) Kill(pid int32, force bool) error {
	p, err := process.NewProcess(pid)
	if err != nil {
		return err
	}

	if force {
		// 🔪 强制击杀 (SIGKILL) - 进程没机会留遗言
		return p.Kill()
	}

	// 🏳️ 优雅请求 (SIGTERM) - 进程可以捕获并清理
	return p.Terminate()
}

// getName 辅助函数：获取进程名
func (l *LocalProvider) getName(p *process.Process) string {
	// 1. 尝试获取标准名称
	name, _ := p.Name()
	if name != "" {
		return name
	}

	// 2. 尝试获取执行路径的基础名
	exe, _ := p.Exe()
	if exe != "" {
		return filepath.Base(exe)
	}

	// 3. 尝试命令行
	cmdline, _ := p.Cmdline()
	if cmdline != "" {
		parts := strings.Fields(cmdline)
		if len(parts) > 0 {
			return filepath.Base(parts[0])
		}
	}

	// 🔴 修改这里：如果都获取不到，说明很可能是权限不足
	// 返回一个提示，或者保留 <Unknown> 但心里有数
	return "<System/Access Denied>"
}
