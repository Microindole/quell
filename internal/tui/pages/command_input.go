package pages

import (
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
)

type CommandInput struct {
	state     *SharedState
	textInput textinput.Model
	err       error
}

func NewCommandInput(state *SharedState) *CommandInput {
	ti := textinput.New()
	ti.Placeholder = "Type a command (e.g. /help)..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 40

	return &CommandInput{
		state:     state,
		textInput: ti,
	}
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
		}
	}

	c.textInput, cmd = c.textInput.Update(msg)
	return c, cmd
}

func (c *CommandInput) executeCommand(cmdStr string) (View, tea.Cmd) {
	if cmdStr == "" {
		return c, nil
	}

	// 1. 解析命令和参数 (例如 "/kill 123" -> cmd="/kill", args=["123"])
	parts := strings.Fields(cmdStr)
	cmdName := parts[0]
	var args []string
	if len(parts) > 1 {
		args = parts[1:]
	}

	// 2. 🔥 查表执行
	if handler, exists := CommandRegistry[cmdName]; exists {
		// 调用注入的函数
		view, cmd := handler(args, c.state)

		// 如果 handler 返回了新的 View (例如 /help 返回 HelpView)，则进行 Replace 跳转
		if view != nil {
			return c, Replace(view)
		}
		// 否则只执行 cmd (例如 /quit)
		return c, cmd
	}

	// 3. 未知命令处理
	c.textInput.SetValue("")
	c.textInput.Placeholder = "Unknown command: " + cmdName
	return c, nil
}

func (c *CommandInput) View() string {
	return "\n\n" + inputBoxStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render("Command Mode"),
			c.textInput.View(),
		),
	) + "\n\n"
}

func (c *CommandInput) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "exec")),
	}
}
