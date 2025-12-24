//go:build windows

package system

import "os"

// IsAdmin Checks if the current process has administrative privileges
func IsAdmin() bool {
	// 尝试打开物理磁盘，只有管理员能做 (一种常见的判定方式)
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	if err == nil {
		return true
	}
	return false
}
