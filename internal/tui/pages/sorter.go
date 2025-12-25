package pages

import "github.com/Microindole/quell/internal/core"

type Sorter interface {
	Name() string
	Less(p1, p2 core.Process) bool
}

type StatusSorter struct{}

func (s StatusSorter) Name() string { return "Status (Paused Top)" }
func (s StatusSorter) Less(p1, p2 core.Process) bool {
	// 逻辑：暂停的进程 (IsSuspended=true) 排在前面
	sus1 := p1.IsSuspended()
	sus2 := p2.IsSuspended()

	if sus1 && !sus2 {
		return true // p1 是暂停的，排前面
	}
	if !sus1 && sus2 {
		return false // p2 是暂停的，排前面
	}

	// 如果状态相同，按 CPU 降序排 (作为二级排序)
	return p1.CpuPercent > p2.CpuPercent
}

type PIDSorter struct{}

func (s PIDSorter) Name() string                  { return "PID ⬆" }
func (s PIDSorter) Less(p1, p2 core.Process) bool { return p1.PID < p2.PID }

type MemSorter struct{}

func (s MemSorter) Name() string                  { return "Memory ⬇" }
func (s MemSorter) Less(p1, p2 core.Process) bool { return p1.MemoryUsage > p2.MemoryUsage }

type CPUSorter struct{}

func (s CPUSorter) Name() string                  { return "CPU ⬇" }
func (s CPUSorter) Less(p1, p2 core.Process) bool { return p1.CpuPercent > p2.CpuPercent }
