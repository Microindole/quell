package commands

import (
	"github.com/Microindole/quell/internal/tui/pages"
)

// RegisterAll 将所有命令注入到提供的注册表中
func RegisterAll(registry map[string]pages.CommandFunc) {
	// 在这里维护命令列表，整洁且集中
	registry["/help"] = HelpCmd
	registry["/quit"] = QuitCmd
	registry["/exit"] = QuitCmd
	registry["/kill"] = KillCmd

	registry["/stop"] = PauseCmd
	registry["/pause"] = PauseCmd
	registry["/cont"] = ResumeCmd
	registry["/resume"] = ResumeCmd
}
