package core

type Provider interface {
	ListProcesses() ([]Process, error)
	Kill(pid int32, force bool) error
	Suspend(pid int32) error
	Resume(pid int32) error
}
