package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type PausedProcess struct {
	PID        int32 `json:"pid"`
	CreateTime int64 `json:"create_time"`
}

// Config 定义我们需要保存的字段
type Config struct {
	SortIndex   int             `json:"sort_index"` // 排序方式索引
	TreeMode    bool            `json:"tree_mode"`  // 是否开启树状图
	PausedProcs []PausedProcess `json:"paused_procs"`
}

// Manager 配置管理器
type Manager struct {
	configPath string
	mu         sync.Mutex
}

// NewManager 创建管理器，自动定位到 ~/.quell/config.json
func NewManager() *Manager {
	home, _ := os.UserHomeDir()
	configDir := filepath.Join(home, ".quell")

	// 确保目录存在
	_ = os.MkdirAll(configDir, 0755)

	return &Manager{
		configPath: filepath.Join(configDir, "config.json"),
	}
}

// Load 读取配置，如果文件不存在则返回默认值
func (m *Manager) Load() (*Config, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	f, err := os.Open(m.configPath)
	if err != nil {
		// 文件不存在，返回默认配置
		return &Config{
			SortIndex:   0,
			TreeMode:    false,
			PausedProcs: []PausedProcess{},
		}, nil
	}
	defer func() { _ = f.Close() }()

	var cfg Config
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Save 保存配置到磁盘
func (m *Manager) Save(cfg *Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	f, err := os.Create(m.configPath)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := f.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ") // 美化输出
	return encoder.Encode(cfg)
}
