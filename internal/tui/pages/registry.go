package pages

import tea "github.com/charmbracelet/bubbletea"

// CommandFunc 定义命令函数的标准签名
// args: 参数列表 (例如 ["-f"]), state: 全局状态
type CommandFunc func(args []string, state *SharedState) (View, tea.Cmd)

// CommandRegistry 全局命令注册表
// 初始化为空，等待 main/model 层注入具体实现
var CommandRegistry = make(map[string]CommandFunc)
