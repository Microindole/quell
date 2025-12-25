package pages

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var warningStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#1A1A1A")).
	Background(lipgloss.Color("#FFA500")).
	Padding(1, 2).
	Bold(true)

type ConfirmDialog struct {
	message   string
	onConfirm tea.Cmd
}

func NewConfirmDialog(msg string, onConfirm tea.Cmd) *ConfirmDialog {
	return &ConfirmDialog{message: msg, onConfirm: onConfirm}
}

func (c *ConfirmDialog) Init() tea.Cmd { return nil }

func (c *ConfirmDialog) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y", "enter":
			return c, tea.Batch(Pop(), c.onConfirm)
		default:
			return c, Pop()
		}
	}
	return c, nil
}

func (c *ConfirmDialog) View() string {
	return "\n\n" + warningStyle.Render("⚠️  "+c.message+" (y/N)") + "\n\n"
}

func (c *ConfirmDialog) ShortHelp() []key.Binding { return nil }
