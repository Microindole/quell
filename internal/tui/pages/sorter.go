package pages

import "github.com/Microindole/quell/internal/core"

type Sorter interface {
	Name() string
	Less(p1, p2 core.Process) bool
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
