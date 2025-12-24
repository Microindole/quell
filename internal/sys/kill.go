package sys

import (
	"fmt"

	"github.com/shirou/gopsutil/v3/process"
)

// KillProcess 根据 PID 终止进程
func KillProcess(pid int32) error {
	p, err := process.NewProcess(pid)
	if err != nil {
		return fmt.Errorf("process not found: %v", err)
	}

	// Kill() 发送 SIGKILL 信号 (强制终止)
	return p.Kill()
}
