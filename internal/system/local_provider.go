package system

import (
	"path/filepath"
	"sync"

	"github.com/Microindole/quell/internal/core"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

type LocalProvider struct {
	mu        sync.Mutex
	procCache map[int32]*process.Process
}

func NewLocalProvider() *LocalProvider {
	return &LocalProvider{
		procCache: make(map[int32]*process.Process),
	}
}

// ListProcesses 获取全量进程列表
func (l *LocalProvider) ListProcesses() ([]core.Process, error) {
	// 1. 获取所有运行中的进程 ID (不再局限于 TCP 连接)
	pids, err := process.Pids()
	if err != nil {
		return nil, err
	}

	// 2. 预取所有 TCP 连接信息，建立 PID -> Port 的映射索引
	// 这样就不用对每个进程都去查一次网络，极大提升性能
	portMap := make(map[int32]int)
	if conns, err := net.Connections("tcp"); err == nil {
		for _, c := range conns {
			if c.Status == "LISTEN" && c.Pid > 0 {
				portMap[c.Pid] = int(c.Laddr.Port)
			}
		}
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	var results []core.Process
	seenPids := make(map[int32]bool)

	for _, pid := range pids {
		seenPids[pid] = true

		// --- 缓存复用逻辑 (解决 CPU 0% 问题) ---
		proc, exists := l.procCache[pid]
		if !exists {
			newProc, err := process.NewProcess(pid)
			if err != nil {
				continue // 进程可能刚结束
			}
			proc = newProc
			l.procCache[pid] = proc
		}

		// --- 核心过滤逻辑 ---
		// 尝试获取名字，如果失败（Access Denied），说明我们没权限看它
		// 直接 continue 跳过，不显示在列表中
		name, err := proc.Name()
		if err != nil || name == "" {
			continue
		}

		// 获取其他信息
		cpuPercent, _ := proc.Percent(0)
		memInfo, _ := proc.MemoryInfo()
		var memUsage uint64
		if memInfo != nil {
			memUsage = memInfo.RSS
		}

		user, _ := proc.Username()

		// 组装名称 (辅助函数优化显示)
		displayName := l.refineName(proc, name)

		results = append(results, core.Process{
			PID:         pid,
			Name:        displayName,
			Port:        portMap[pid], // 如果该进程有监听端口，这里会自动填上，否则是 0
			Protocol:    "TCP",        // 默认 TCP
			Cmdline:     l.getCmdlineSafe(proc),
			MemoryUsage: memUsage,
			CpuPercent:  cpuPercent,
			User:        user,
		})
	}

	// 🧹 清理已退出的进程缓存 (防止内存泄漏)
	for cachedPid := range l.procCache {
		if !seenPids[cachedPid] {
			delete(l.procCache, cachedPid)
		}
	}

	return results, nil
}

func (l *LocalProvider) Kill(pid int32, force bool) error {
	p, err := process.NewProcess(pid)
	if err != nil {
		return err
	}
	if force {
		return p.Kill()
	}
	return p.Terminate()
}

// 辅助：获取更友好的进程名
func (l *LocalProvider) refineName(p *process.Process, rawName string) string {
	if rawName != "" {
		return rawName
	}
	exe, _ := p.Exe()
	if exe != "" {
		return filepath.Base(exe)
	}
	return "Unknown"
}

// 辅助：安全获取命令行，失败返回空
func (l *LocalProvider) getCmdlineSafe(p *process.Process) string {
	cmd, _ := p.Cmdline()
	return cmd
}
