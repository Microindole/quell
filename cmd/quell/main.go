package main

import (
	"github.com/Microindole/quell/internal/config"
	"github.com/Microindole/quell/internal/core"
	"github.com/Microindole/quell/internal/system"
	"github.com/Microindole/quell/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// 1. 加载配置
	cfgManager := config.NewManager()
	cfg, _ := cfgManager.Load() // 忽略错误使用默认值

	// 2. 初始化 Service
	provider := system.NewLocalProvider()
	service := core.NewService(provider)

	// 3. 🔥 核心修正：恢复暂停状态（带类型转换）
	// 因为 Service 为了解耦使用了匿名结构体，这里需要手动转换一下
	if len(cfg.PausedProcs) > 0 {
		// 定义一个临时的匿名结构体切片，符合 Service.RestorePausedPIDs 的签名
		var restoreList []struct {
			PID        int32
			CreateTime int64
		}

		for _, p := range cfg.PausedProcs {
			restoreList = append(restoreList, struct {
				PID        int32
				CreateTime int64
			}{PID: p.PID, CreateTime: p.CreateTime})
		}

		service.RestorePausedPIDs(restoreList)
	}

	// 4. 启动 UI
	model := tui.NewModel(service, cfg)
	p := tea.NewProgram(model, tea.WithAltScreen())

	// ... Run ...

	// 5. 退出保存
	if _, err := p.Run(); err == nil {
		finalConfig := model.GetSnapshot()
		_ = cfgManager.Save(finalConfig)
	}
}
