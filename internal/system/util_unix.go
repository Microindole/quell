//go:build !windows

package system

import (
	"os"

	"github.com/shirou/gopsutil/v3/process"
)

func IsAdmin() bool {
	return os.Geteuid() == 0
}

func GetProcessStatus(p *process.Process) string {
	status, err := p.Status()
	if err != nil || len(status) == 0 {
		return "?"
	}
	return status[0] // Unix 返回的是切片，取第一个
}
