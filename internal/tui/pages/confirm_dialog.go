package pages

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	// 1. 边框颜色
	borderColor = lipgloss.Color("#FFA500") // 橙色

	// 2. 弹窗外框 (去除顶部边框，改用 Title 覆盖)
	dialogBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(1, 0).    // 左右不留白，让标题栏填满
			BorderTop(false). // 去掉上边框，用实心标题栏代替
			Width(50).
			Align(lipgloss.Center)

	// 3. 实心标题栏 (解决对齐和美观问题)
	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1A1A1A")). // 深色文字
			Background(borderColor).               // 橙色背景
			Bold(true).
			Width(50). // 填满宽度
			Align(lipgloss.Center)

	// 4. 内容文字
	textStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Bold(true).
			Padding(1, 2) // 内部留白

	// 5. 底部提示
	subTextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			PaddingBottom(1)
)

type ConfirmDialog struct {
	message   string
	onConfirm tea.Cmd
	width     int
	height    int
}

func NewConfirmDialog(msg string, onConfirm tea.Cmd) *ConfirmDialog {
	return &ConfirmDialog{message: msg, onConfirm: onConfirm}
}

func (c *ConfirmDialog) Init() tea.Cmd { return nil }

func (c *ConfirmDialog) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.width = msg.Width
		c.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		// 1. 确认操作 (Yes)
		case "y", "Y", "enter":
			return c, tea.Batch(Pop(), c.onConfirm)

		// 2. 拒绝/取消操作 (No)
		// 明确指定哪些键触发取消，而不是用 default
		case "n", "N", "esc", "q", "backspace":
			return c, Pop()

		// 3. 强制退出程序
		case "ctrl+c":
			return c, tea.Quit

		// 4. 忽略其他按键
		// 这里包括：单独的 Ctrl/Alt/Shift、鼠标滚轮、F1-F12 等
		// 不做任何状态变更 (return c, nil)，保持弹窗显示
		default:
			return c, nil
		}
	}
	return c, nil
}

func (c *ConfirmDialog) View() string {
	// 渲染标题栏 (不用 Emoji，用纯文字保证绝对居中)
	header := headerStyle.Render("WARNING")

	// 渲染主体内容
	content := lipgloss.JoinVertical(lipgloss.Center,
		textStyle.Render(c.message),
		subTextStyle.Render("(y/N)"),
	)

	// 组合：实心标题 + 边框内容
	// 注意：因为我们去掉了 BorderTop，所以要把 Header 拼在最上面
	// 为了让圆角闭合，这里用了一个小技巧：
	// 既然 lipgloss 的 BorderTop(false) 会导致上方开口，
	// 我们不如恢复 BorderTop(true)，但把 Header 放在 Box 内部的最上方。

	// 重新调整样式以适应内部 Header
	realBoxStyle := dialogBoxStyle.Copy().
		BorderTop(true).
		Padding(0) // 清除 padding 以便 header 贴边

	ui := realBoxStyle.Render(
		lipgloss.JoinVertical(lipgloss.Center,
			header,  // 橙色条在框内顶部
			content, // 白色文字在下面
		),
	)

	// 绝对居中
	if c.width > 0 && c.height > 0 {
		// 减去 2 行高度预留给底部的 Status Bar
		// 这样弹窗会居中于剩余空间，而不会覆盖到底部
		safeHeight := c.height - 2
		if safeHeight < 0 {
			safeHeight = 0
		}

		return lipgloss.Place(
			c.width, safeHeight,
			lipgloss.Center, lipgloss.Center,
			ui,
		)
	}

	return "\n\n" + lipgloss.PlaceHorizontal(80, lipgloss.Center, ui)
}

func (c *ConfirmDialog) ShortHelp() []key.Binding { return nil }
