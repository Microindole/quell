package core

import "fmt"

// Process item for the list
type Process struct {
	PID      int32
	Name     string
	Port     int    // 如果是端口模式
	Protocol string // TCP/UDP
}

// 必须实现 list.Item 接口 (为了配合 bubbles/list 组件)
func (p Process) FilterValue() string { return p.Name }
func (p Process) Title() string       { return p.Name }
func (p Process) Description() string {
	return fmt.Sprintf("PID: %d | Port: %d (%s)", p.PID, p.Port, p.Protocol)
}
