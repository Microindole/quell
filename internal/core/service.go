package core

import (
	"sync"
)

type Service struct {
	provider   Provider
	mu         sync.Mutex
	pausedPids map[int32]int64
}

func NewService(p Provider) *Service {
	return &Service{
		provider:   p,
		pausedPids: make(map[int32]int64),
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

	alivePids := make(map[int32]bool)

	for i := range procs {
		pid := procs[i].PID
		alivePids[pid] = true

		// 🔥 核心校验：只有 PID 相同 且 创建时间相同，才认为是“那个被暂停的进程”
		if savedTime, ok := s.pausedPids[pid]; ok {
			if savedTime == procs[i].CreateTime {
				procs[i].Status = "Suspended" // 身份核验通过，标记为暂停
			} else {
				// PID 相同但时间不同 -> 这是个新进程 (PID Reuse)
				// 可以在这里静默移除旧记录，或者留给下面的清理逻辑
				delete(s.pausedPids, pid)
			}
		}
	}
	// 清理逻辑：如果 PID 根本就不存在了，删掉
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
		// 🔥 暂停成功后，获取该进程的“身份证” (CreateTime)
		ct, ctErr := s.provider.GetCreateTime(pid)
		if ctErr == nil {
			s.mu.Lock()
			s.pausedPids[pid] = ct // 记录 PID + 时间
			s.mu.Unlock()
		}
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

// RestorePausedPIDs 启动时调用：恢复暂停列表
func (s *Service) RestorePausedPIDs(list []struct {
	PID        int32
	CreateTime int64
}) { // 或使用 config.PausedProcess 类型
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range list {
		s.pausedPids[item.PID] = item.CreateTime
	}
}

// GetPausedPIDs 退出时调用：获取当前所有暂停的 PID
func (s *Service) GetPausedPIDs() []int32 {
	s.mu.Lock()
	defer s.mu.Unlock()

	var list []int32
	for pid := range s.pausedPids {
		list = append(list, pid)
	}
	return list
}

func (s *Service) GetPausedProcs() []struct {
	PID        int32
	CreateTime int64
} {
	s.mu.Lock()
	defer s.mu.Unlock()
	var list []struct {
		PID        int32
		CreateTime int64
	}
	for pid, ct := range s.pausedPids {
		list = append(list, struct {
			PID        int32
			CreateTime int64
		}{PID: pid, CreateTime: ct})
	}
	return list
}

func (s *Service) GetConnections(pid int32) ([]Connection, error) {
	return s.provider.GetConnections(pid)
}
