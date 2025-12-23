package sys

import (
	"path/filepath"
	"strings"

	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"

	// 👇 替换为你的模块名
	"github.com/Microindole/quell/internal/domain"
)

func GetProcesses() ([]domain.Process, error) {
	conns, err := net.Connections("tcp")
	if err != nil {
		return nil, err
	}

	var results []domain.Process

	for _, conn := range conns {
		if conn.Status != "LISTEN" {
			continue
		}

		// 1. 特判 Windows 系统核心进程
		if conn.Pid == 0 {
			continue // PID 0 通常不占用端口，或者是 System Idle
		}
		if conn.Pid == 4 {
			results = append(results, domain.Process{
				PID:      conn.Pid,
				Name:     "System", // 手动命名
				Port:     int(conn.Laddr.Port),
				Protocol: "TCP",
			})
			continue
		}

		p, err := process.NewProcess(conn.Pid)
		if err != nil {
			continue
		}

		// 2. 尝试获取名字 (多重策略)
		name := getName(p)

		results = append(results, domain.Process{
			PID:      conn.Pid,
			Name:     name,
			Port:     int(conn.Laddr.Port),
			Protocol: "TCP",
		})
	}

	return results, nil
}

// 辅助函数：尽力获取进程名
func getName(p *process.Process) string {
	// 策略 1: 标准 Name()
	name, err := p.Name()
	if err == nil && name != "" {
		return name
	}

	// 策略 2: 获取执行路径的文件名 (比如 D:\Soft\app.exe -> app.exe)
	exe, err := p.Exe()
	if err == nil && exe != "" {
		return filepath.Base(exe)
	}

	// 策略 3: 获取命令行启动参数的第一个 (比如 ./app run -> ./app)
	cmdline, err := p.Cmdline()
	if err == nil && cmdline != "" {
		// 简单处理：取空格前的部分作为名字
		parts := strings.Fields(cmdline)
		if len(parts) > 0 {
			return filepath.Base(parts[0])
		}
	}

	// 策略 4: 实在拿不到，说明是权限受限的系统进程
	return "<System Process>"
}
