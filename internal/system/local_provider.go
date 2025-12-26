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
	procCache map[int32]cachedProcess
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
		var currentCreateTime int64 // 🔥 新增变量，用于暂存时间

		// --- 智能缓存逻辑 ---
		cached, exists := l.procCache[pid]

		if exists {
			ct, err := cached.proc.CreateTime()
			// 验证时间是否一致
			if err == nil && ct == cached.createTime {
				proc = cached.proc
				currentCreateTime = cached.createTime // 命中缓存，取缓存时间
			} else {
				exists = false
			}
		}

		if !exists {
			newProc, err := process.NewProcess(pid)
			if err != nil {
				continue
			}
			ct, _ := newProc.CreateTime()
			proc = newProc
			currentCreateTime = ct // 🔥 新进程，取刚获取的时间

			// 更新缓存
			l.procCache[pid] = cachedProcess{
				proc:       newProc,
				createTime: ct,
			}
		}
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
			Ports:       portMap[pid], // 这里现在是 []int
			Protocol:    "TCP",
			Cmdline:     l.getCmdlineSafe(proc),
			MemoryUsage: memUsage,
			CpuPercent:  cpuPercent,
			User:        user,
			Status:      statusStr,
			CreateTime:  currentCreateTime,
		})
	}

	// 清理缓存
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

func (l *LocalProvider) GetCreateTime(pid int32) (int64, error) {
	p, err := process.NewProcess(pid)
	if err != nil {
		return 0, err
	}
	return p.CreateTime()
}

func (l *LocalProvider) GetConnections(pid int32) ([]core.Connection, error) {
	p, err := process.NewProcess(pid)
	if err != nil {
		return nil, err
	}

	// 获取该进程的所有网络连接
	conns, err := p.Connections()
	if err != nil {
		return []core.Connection{}, nil
	}

	var results []core.Connection
	for _, c := range conns {
		results = append(results, core.Connection{
			Fd:         c.Fd,
			Family:     c.Family,
			Type:       c.Type,
			LocalIP:    c.Laddr.IP,
			LocalPort:  int(c.Laddr.Port),
			RemoteIP:   c.Raddr.IP,
			RemotePort: int(c.Raddr.Port),
			Status:     c.Status,
		})
	}
	return results, nil
}
