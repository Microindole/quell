package core

type Service struct {
	provider Provider
}

func NewService(p Provider) *Service {
	return &Service{provider: p}
}

// GetProcesses 获取进程列表
func (s *Service) GetProcesses() ([]Process, error) {
	// 这里以后可以加缓存逻辑，或者白名单过滤
	return s.provider.ListProcesses()
}

// Kill 终止进程
func (s *Service) Kill(pid int32, force bool) error {
	return s.provider.Kill(pid, force)
}
