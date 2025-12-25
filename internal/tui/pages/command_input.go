package pages

import (
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	inputBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1).
			Width(50)
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true)

	suggestionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))
	activeSuggestionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFDF5")).
				Background(lipgloss.Color("#7D56F4")).
				Padding(0, 1)
)

type CommandInput struct {
	state     *SharedState
	textInput textinput.Model
	err       error
	matches   []string // 当前匹配的命令列表
	matchIdx  int      // 当前选中的补全项索引 (-1 表示未选中)
}

func NewCommandInput(state *SharedState, initialText string) *CommandInput {
	ti := textinput.New()
	ti.Placeholder = "Type a command (e.g. /help)..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 40

	ti.SetValue(initialText)

	// 初始化并立即计算一次匹配 (处理带初始值的情况 /pkill)
	c := &CommandInput{
		state:     state,
		textInput: ti,
		matchIdx:  -1,
	}
	c.updateMatches(initialText)
	return c
}

func (c *CommandInput) Init() tea.Cmd {
	return textinput.Blink
}

func (c *CommandInput) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			return c, Pop()
		case tea.KeyEnter:
			value := strings.TrimSpace(c.textInput.Value())
			return c.executeCommand(value)

		// 🔥 新增：Tab 键循环补全
		case tea.KeyTab:
			if len(c.matches) > 0 {
				// 1. 循环索引
				c.matchIdx = (c.matchIdx + 1) % len(c.matches)

				// 2. 填入选中的命令
				selection := c.matches[c.matchIdx]
				c.textInput.SetValue(selection)

				// 3. 将光标移到末尾，方便继续输入参数
				c.textInput.SetCursor(len(selection))

				// 阻止 Tab 键传递给 textInput (避免焦点问题)
				return c, nil
			}
		}
	}

	c.textInput, cmd = c.textInput.Update(msg)

	// 🔥 每次输入变化后，刷新匹配列表
	// 注意：如果是 Tab 键触发的 Update，已经在上面 return 了，所以不会执行这里
	// 这正好符合逻辑：用户手动输入时刷新列表并重置索引；用户 Tab 循环时保持列表不变。
	currentVal := c.textInput.Value()
	c.updateMatches(currentVal)

	return c, cmd
}

// updateMatches 根据当前输入更新候选列表
func (c *CommandInput) updateMatches(input string) {
	// 如果输入为空，清空建议
	if input == "" {
		c.matches = nil
		c.matchIdx = -1
		return
	}

	var results []string
	inputLower := strings.ToLower(input)

	// 遍历全局命令注册表
	for cmd := range CommandRegistry {
		// 简单的这个前缀匹配：输入 /k -> 匹配 /kill, /killall
		if strings.HasPrefix(strings.ToLower(cmd), inputLower) {
			results = append(results, cmd)
		}
	}

	// 排序保证顺序稳定
	sort.Strings(results)

	c.matches = results
	c.matchIdx = -1 // 重置选中状态
}

func (c *CommandInput) executeCommand(cmdStr string) (View, tea.Cmd) {
	if cmdStr == "" {
		return c, nil
	}

	parts := strings.Fields(cmdStr)
	cmdName := parts[0]
	var args []string
	if len(parts) > 1 {
		args = parts[1:]
	}

	if handler, exists := CommandRegistry[cmdName]; exists {
		view, cmd := handler(args, c.state)
		if view != nil {
			return c, Replace(view)
		}
		return c, cmd
	}

	c.textInput.SetValue("")
	c.textInput.Placeholder = "Unknown command: " + cmdName
	return c, nil
}

func (c *CommandInput) View() string {
	// 🔥 构建建议列表视图
	var suggestionsView string
	if len(c.matches) > 0 {
		var items []string
		for i, m := range c.matches {
			if i == c.matchIdx {
				// 高亮选中的
				items = append(items, activeSuggestionStyle.Render(m))
			} else {
				// 普通样式
				items = append(items, suggestionStyle.Render(m))
			}
		}
		// 用空格分隔横向排列
		suggestionsView = strings.Join(items, "  ")
	} else if c.textInput.Value() != "" {
		// 如果输入了内容但没有匹配项 (可选)
		// suggestionsView = suggestionStyle.Render("(no matches)")
	}

	// 垂直拼接：标题 -> 输入框 -> 建议列表
	ui := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("Command Mode"),
		c.textInput.View(),
		suggestionsView, // 放在输入框下方
	)

	return "\n\n" + inputBoxStyle.Render(ui) + "\n\n"
}

func (c *CommandInput) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "complete")), // 提示 Tab 键
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "exec")),
	}
}
