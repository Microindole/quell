package pages

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var helpBoxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("#25A065")).
	Padding(1, 2)

type HelpView struct{}

func NewHelpView() *HelpView {
	return &HelpView{}
}

func (h *HelpView) Init() tea.Cmd { return nil }

func (h *HelpView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// 任意键退出
		if msg.String() == "esc" || msg.String() == "enter" || msg.String() == "q" {
			return h, Pop()
		}
	}
	return h, nil
}

func (h *HelpView) View() string {
	content := `
Quell - Help

Global Keys:
  Ctrl+C      : Quit

List View:
  /           : Filter processes
  x           : Kill process
  X           : Force kill process
  enter/space : Inspect process details
  tab         : Sort (PID/Mem/CPU)
  t           : Toggle Tree View
  ` + "`" + `           : Command Mode

Commands (type after pressing ` + "`" + `):
  /help       : Show this help
  /quit       : Exit application
`
	return "\n" + helpBoxStyle.Render(content) + "\n"
}

func (h *HelpView) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	}
}
