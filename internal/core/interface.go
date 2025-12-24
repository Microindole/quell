package core

type Provider interface {
	ListProcesses() ([]Process, error)
	Kill(pid int32, force bool) error
}
