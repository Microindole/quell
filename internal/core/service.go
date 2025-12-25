package core

import (
	"sync"
)

type Service struct {
	provider Provider
	// 🔥 新增：本地维护的暂停名单
	mu         sync.Mutex
	pausedPids map[int32]bool
}

func NewService(p Provider) *Service {
	return &Service{
		provider:   p,
		pausedPids: make(map[int32]bool), // 初始化 map
	}
}

// GetProcesses 获取进程列表
func (s *Service) GetProcesses() ([]Process, error) {
	procs, err := s.provider.ListProcesses()
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 用于记录当前存活的 PID，以便清理僵尸记录
	alivePids := make(map[int32]bool)

	for i := range procs {
		pid := procs[i].PID
		alivePids[pid] = true
		if s.pausedPids[pid] {
			procs[i].Status = "Suspended" // 强制标记为暂停
		}
	}
	for pid := range s.pausedPids {
		if !alivePids[pid] {
			delete(s.pausedPids, pid)
		}
	}

	return procs, nil
}

// Kill 终止进程
func (s *Service) Kill(pid int32, force bool) error {
	// 如果进程被杀，理论上 GetProcesses 的清理逻辑会处理，
	// 但为了保险，这里也可以直接移除
	err := s.provider.Kill(pid, force)
	if err == nil {
		s.mu.Lock()
		delete(s.pausedPids, pid)
		s.mu.Unlock()
	}
	return err
}

func (s *Service) Suspend(pid int32) error {
	err := s.provider.Suspend(pid)
	if err == nil {
		// 🔥 成功暂停后，加入名单
		s.mu.Lock()
		s.pausedPids[pid] = true
		s.mu.Unlock()
	}
	return err
}

func (s *Service) Resume(pid int32) error {
	err := s.provider.Resume(pid)
	if err == nil {
		// 🔥 成功恢复后，移出名单
		s.mu.Lock()
		delete(s.pausedPids, pid)
		s.mu.Unlock()
	}
	return err
}
