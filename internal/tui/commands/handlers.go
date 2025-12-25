package commands

import (
	"strconv"

	"github.com/Microindole/quell/internal/tui/pages"
	tea "github.com/charmbracelet/bubbletea"
)

// HelpCmd 显示帮助页面
func HelpCmd(args []string, state *pages.SharedState) (pages.View, tea.Cmd) {
	// 引用 pages 包创建视图
	return pages.NewHelpView(), nil
}

// QuitCmd 退出程序
func QuitCmd(args []string, state *pages.SharedState) (pages.View, tea.Cmd) {
	return nil, tea.Quit
}

// KillCmd 实现 /kill <pid>
func KillCmd(args []string, state *pages.SharedState) (pages.View, tea.Cmd) {
	if len(args) == 0 {
		return nil, nil
	} // 简化错误处理
	pid, err := strconv.ParseInt(args[0], 10, 32)
	if err != nil {
		return nil, nil
	}

	cmd := func() tea.Msg {
		err := state.Service.Kill(int32(pid), false)
		return pages.ProcessActionMsg{Err: err, Action: "Killed"}
	}
	return nil, tea.Batch(pages.Pop(), cmd)
}

func PauseCmd(args []string, state *pages.SharedState) (pages.View, tea.Cmd) {
	if len(args) == 0 {
		return nil, nil
	}
	pid, err := strconv.ParseInt(args[0], 10, 32)
	if err != nil {
		return nil, nil
	}

	cmd := func() tea.Msg {
		err := state.Service.Suspend(int32(pid))
		return pages.ProcessActionMsg{Err: err, Action: "Suspended"}
	}
	return nil, tea.Batch(pages.Pop(), cmd)
}

func ResumeCmd(args []string, state *pages.SharedState) (pages.View, tea.Cmd) {
	if len(args) == 0 {
		return nil, nil
	}
	pid, err := strconv.ParseInt(args[0], 10, 32)
	if err != nil {
		return nil, nil
	}

	cmd := func() tea.Msg {
		err := state.Service.Resume(int32(pid))
		return pages.ProcessActionMsg{Err: err, Action: "Resumed"}
	}
	return nil, tea.Batch(pages.Pop(), cmd)
}
