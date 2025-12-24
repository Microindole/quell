//go:build !windows

package system

import "os"

func IsAdmin() bool {
	// Unix/Linux 下，root 用户的 EUID 为 0
	return os.Geteuid() == 0
}
