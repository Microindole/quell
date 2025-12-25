package system

import (
	"path/filepath"
	"sort"
	"sync"

	"github.com/Microindole/quell/internal/core"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// 定义缓存项，包含创建时间用于验证 PID 是否被复用
type cachedProcess struct {
	proc       *process.Process
	createTime int64
}

type LocalProvider struct {
	mu        sync.Mutex
	procCache map[int32]cachedProcess // 👈 修改：使用结构体存储
}

func NewLocalProvider() *LocalProvider {
	return &LocalProvider{
		procCache: make(map[int32]cachedProcess),
	}
}

// ListProcesses 获取全量进程列表
func (l *LocalProvider) ListProcesses() ([]core.Process, error) {
	// 1. 获取所有 PID
	pids, err := process.Pids()
	if err != nil {
		return nil, err
	}

	// 2. 预取网络连接 (允许失败，失败则端口为空)
	// 使用 map[int32][]int 来存储每个 PID 的多个端口
	portMap := make(map[int32][]int)
	if conns, err := net.Connections("tcp"); err == nil {
		for _, c := range conns {
			if c.Status == "LISTEN" && c.Pid > 0 {
				portMap[c.Pid] = append(portMap[c.Pid], int(c.Laddr.Port))
			}
		}
	}
	// 对端口进行去重和排序，为了显示美观
	for pid, ports := range portMap {
		portMap[pid] = uniqueSortedPorts(ports)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	var results []core.Process
	seenPids := make(map[int32]bool)

	for _, pid := range pids {
		seenPids[pid] = true

		var proc *process.Process

		// --- 智能缓存逻辑 (解决 PID 复用 + CPU 0% 问题) ---
		cached, exists := l.procCache[pid]

		if exists {
			// 验证该 PID 是否还是原来的进程 (通过创建时间判断)
			// 注意：Process.CreateTime() 会实时获取当前 PID 的创建时间
			currentCreateTime, err := cached.proc.CreateTime()
			if err == nil && currentCreateTime == cached.createTime {
				// 是一样的进程，安全复用
				proc = cached.proc
			} else {
				// PID 被复用，或者是第一次获取 CreateTime 失败，视为新进程
				exists = false
			}
		}

		if !exists {
			newProc, err := process.NewProcess(pid)
			if err != nil {
				continue // 进程可能刚结束
			}
			ct, _ := newProc.CreateTime() // 获取创建时间
			proc = newProc

			// 更新缓存
			l.procCache[pid] = cachedProcess{
				proc:       newProc,
				createTime: ct,
			}
		}
		// ---------------------------------------------

		// 过滤系统进程/无权限进程
		name, err := proc.Name()
		if err != nil || name == "" {
			continue
		}

		ppid, _ := proc.Ppid()

		// 获取动态数据
		cpuPercent, _ := proc.Percent(0)
		memInfo, _ := proc.MemoryInfo()
		var memUsage uint64
		if memInfo != nil {
			memUsage = memInfo.RSS // RSS 通常对应 Task Manager 的工作集
		}

		user, _ := proc.Username()

		statusStr := GetProcessStatus(proc)

		results = append(results, core.Process{
			PID:         pid,
			PPID:        ppid,
			Name:        l.refineName(proc, name),
			Ports:       portMap[pid], // 👈 这里现在是 []int
			Protocol:    "TCP",
			Cmdline:     l.getCmdlineSafe(proc),
			MemoryUsage: memUsage,
			CpuPercent:  cpuPercent,
			User:        user,
			Status:      statusStr,
		})
	}

	// 🧹 清理缓存
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

func (l *LocalProvider) getCmdlineSafe(p *process.Process) string {
	cmd, _ := p.Cmdline()
	return cmd
}

// 辅助：端口去重排序
func uniqueSortedPorts(ports []int) []int {
	if len(ports) == 0 {
		return nil
	}
	unique := make(map[int]bool)
	var result []int
	for _, p := range ports {
		if !unique[p] {
			unique[p] = true
			result = append(result, p)
		}
	}
	sort.Ints(result)
	return result
}

// Suspend 暂停进程
func (l *LocalProvider) Suspend(pid int32) error {
	p, err := process.NewProcess(pid)
	if err != nil {
		return err
	}
	return p.Suspend()
}

// Resume 恢复进程
func (l *LocalProvider) Resume(pid int32) error {
	p, err := process.NewProcess(pid)
	if err != nil {
		return err
	}
	return p.Resume()
}
