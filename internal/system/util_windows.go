//go:build windows

package system

import (
	"os"

	"github.com/shirou/gopsutil/v3/process"
)

func IsAdmin() bool {
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	return err == nil
}

func GetProcessStatus(p *process.Process) string {
	status, err := p.Status()
	if err != nil || len(status) == 0 {
		return "Running"
	}
	return status[0]
}
