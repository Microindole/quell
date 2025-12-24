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

		// Windows 系统进程特殊处理
		if conn.Pid == 0 {
			continue
		}
		if conn.Pid == 4 {
			results = append(results, core.Process{
				PID:      conn.Pid,
				Name:     "System",
				Port:     int(conn.Laddr.Port),
				Protocol: "TCP",
			})
			continue
		}

		p, err := process.NewProcess(conn.Pid)
		if err != nil {
			continue
		}

		results = append(results, core.Process{
			PID:      conn.Pid,
			Name:     l.getName(p),
			Port:     int(conn.Laddr.Port),
			Protocol: "TCP",
		})
	}

	return results, nil
}

// Kill 杀进程
func (l *LocalProvider) Kill(pid int32) error {
	p, err := process.NewProcess(pid)
	if err != nil {
		return err
	}
	return p.Kill()
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
