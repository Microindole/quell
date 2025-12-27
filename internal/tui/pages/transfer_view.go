package pages

import (
	"fmt"

	"github.com/Microindole/quell/internal/transfer"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/skip2/go-qrcode"
)

// 移除 transferMode 枚举，现在只有一种模式（本地）

type TransferView struct {
	filePath   string
	port       int
	ips        []string
	currentIdx int

	qrCode     string
	stopServer func()
	err        error
}

func NewTransferView(filePath string) *TransferView {
	ips, _ := transfer.GetLocalIPs()
	if len(ips) == 0 {
		ips = []string{"127.0.0.1"}
	}

	port, stop, err := transfer.ServeFile(filePath)

	tv := &TransferView{
		filePath:   filePath,
		port:       port,
		ips:        ips,
		stopServer: stop,
		err:        err,
		// 移除 spinner 和 mode 初始化
	}

	tv.refreshQRCode()
	return tv
}

func (t *TransferView) refreshQRCode() {
	// 只生成本地局域网链接
	var url string
	if len(t.ips) > 0 {
		url = fmt.Sprintf("http://%s:%d", t.ips[t.currentIdx], t.port)
	}

	if url != "" {
		qr, _ := qrcode.New(url, qrcode.Medium)
		t.qrCode = qr.ToSmallString(false)
	}
}

// 移除 uploadCmd 和 uploadFinishedMsg

func (t *TransferView) Init() tea.Cmd {
	return nil // 不再需要 spinner.Tick
}

func (t *TransferView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			if t.stopServer != nil {
				t.stopServer()
			}
			return t, Pop()

		case "tab":
			// 切换 IP
			if len(t.ips) > 1 {
				t.currentIdx = (t.currentIdx + 1) % len(t.ips)
				t.refreshQRCode()
			}

			// 移除 "u" 键的上传逻辑
		}

		// 移除 uploadFinishedMsg 和 spinner.TickMsg 处理
	}
	return t, nil
}

func (t *TransferView) View() string {
	if t.err != nil {
		return fmt.Sprintf("\n  ❌ Error: %v\n\n  (Press Esc to back)", t.err)
	}

	style := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2)

	// 标题更加简洁
	title := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true).Render("🚀 LAN Share")

	// 直接显示局域网分享内容
	currentIP := t.ips[t.currentIdx]
	url := fmt.Sprintf("http://%s:%d", currentIP, t.port)

	content := fmt.Sprintf("\nScan to download (LAN Only):\n%s\n\nURL: %s\n", t.qrCode, url)

	// 底部提示栏简化
	hintText := "\n[Tab] Switch IP"
	if len(t.ips) > 1 {
		hintText += fmt.Sprintf(" (%d/%d)", t.currentIdx+1, len(t.ips))
	}
	hint := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render(hintText)

	content += hint

	return style.Render(lipgloss.JoinVertical(lipgloss.Center, title, content))
}

func (t *TransferView) ShortHelp() []key.Binding {
	keys := []key.Binding{
		key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	}
	if len(t.ips) > 1 {
		keys = append(keys, key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch IP")))
	}
	// 移除 "u" 的帮助信息
	return keys
}
