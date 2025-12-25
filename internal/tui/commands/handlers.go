package commands

import (
	"fmt"
	"strconv"
	"strings"

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

// PKillCmd 实现批量查杀
// 用法：/pkill chrome (杀掉所有名字里包含 chrome 的进程)
func PKillCmd(args []string, state *pages.SharedState) (pages.View, tea.Cmd) {
	// 1. 校验参数
	if len(args) == 0 {
		return nil, func() tea.Msg {
			return pages.ProcessActionMsg{Err: fmt.Errorf("usage: /pkill <name>")}
		}
	}
	target := args[0] // 简单的取第一个参数，例如 "chrome"

	// 2. 执行逻辑
	cmd := func() tea.Msg {
		// 获取最新进程列表
		procs, err := state.Service.GetProcesses()
		if err != nil {
			return pages.ProcessActionMsg{Err: err}
		}

		count := 0
		targetLower := strings.ToLower(target)

		// 遍历并查杀
		for _, p := range procs {
			// 使用 Contains 做模糊匹配 (不区分大小写)
			if strings.Contains(strings.ToLower(p.Name), targetLower) {
				// 执行查杀 (忽略单个失败，只统计成功数)
				if err := state.Service.Kill(p.PID, false); err == nil {
					count++
				}
			}
		}

		// 3. 反馈结果
		if count == 0 {
			return pages.ProcessActionMsg{
				Err: fmt.Errorf("no processes found matching '%s'", target),
			}
		}

		return pages.ProcessActionMsg{
			// 配合 ListView 的 "successfully." 后缀，这里拼接成句子
			// 例如: "Killed 5 processes matching 'chrome'" -> "... successfully."
			Action: fmt.Sprintf("Killed %d processes matching '%s'", count, target),
		}
	}

	// 关闭输入框并执行
	return nil, tea.Batch(pages.Pop(), cmd)
}
