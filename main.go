package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"quell/internal/config"
	"quell/internal/domain"
	"quell/internal/engine"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func runMergeScript(task domain.VideoTask, ffmpegPath string) error {
	// 1. 释放脚本 (这里改个名字，确保每次都用最新的脚本逻辑)
	scriptName := "merge_v3.ps1"
	tmpScript := filepath.Join(os.TempDir(), "quell_"+scriptName)
	// 注意：embed 读取的还是 scripts/merge.ps1
	data, _ := scriptFS.ReadFile("scripts/merge.ps1")
	os.WriteFile(tmpScript, data, 0755)

	// 2. 清洗文件名
	safeTitle := regexp.MustCompile(`[\\/*?:"<>|]`).ReplaceAllString(task.DisplayTitle(), "_")

	// 3. 执行 PowerShell
	// 新增参数: -CoverUrl
	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-File", tmpScript,
		"-TargetDir", task.Dir,
		"-OutputName", safeTitle,
		"-FFmpegPath", ffmpegPath,
		"-CoverUrl", task.Info.CoverUrl,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v | Output: %s", err, string(output))
	}
	if !strings.Contains(string(output), "SUCCESS") {
		return fmt.Errorf("Script executed but no success signal. Output: %s", string(output))
	}
	return nil
}

// --- UI States ---
type sessionState int

const (
	stateConfigDir sessionState = iota // 输入 B站路径
	stateConfigFF                      // 输入 FFmpeg 路径
	stateScanning                      // 扫描中
	stateList                          // 列表显示
)

// --- Model ---
type model struct {
	state     sessionState
	cfg       config.Config
	textInput textinput.Model
	table     table.Model
	spinner   spinner.Model
	tasks     []domain.VideoTask
	err       error
	statusMsg string
}

func initialModel() model {
	ti := textinput.New()
	ti.Focus()
	ti.Width = 60

	s := spinner.New()
	s.Spinner = spinner.Dot

	// 初始化表格样式
	t := table.New(
		table.WithColumns([]table.Column{
			{Title: "状态", Width: 8},
			{Title: "UP主", Width: 15},
			{Title: "标题", Width: 50},
		}),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	m := model{
		textInput: ti,
		table:     t,
		spinner:   s,
	}

	// 尝试加载配置
	loadedCfg, err := config.Load()
	if err == nil && loadedCfg.BiliDir != "" {
		m.cfg = *loadedCfg
		m.state = stateScanning // 有配置直接去扫描
	} else {
		m.state = stateConfigDir // 没配置先去填 B站路径
		m.textInput.Placeholder = "请输入 Bilibili 下载缓存路径 (例如 D:\\Videos\\bilibili)"
	}

	return m
}

func (m model) Init() tea.Cmd {
	if m.state == stateScanning {
		return tea.Batch(m.spinner.Tick, scanCmd(m.cfg.BiliDir))
	}
	return textinput.Blink
}

// --- Commands ---
type scanResultMsg []domain.VideoTask
type processResultMsg struct {
	index int
	err   error
}

func scanCmd(dir string) tea.Cmd {
	return func() tea.Msg {
		tasks, err := engine.Scan(dir)
		if err != nil {
			return err // 简单处理，直接返回 error
		}
		return scanResultMsg(tasks)
	}
}

func processCmd(task domain.VideoTask, ffmpegPath string, idx int) tea.Cmd {
	return func() tea.Msg {
		err := runMergeScript(task, ffmpegPath)
		return processResultMsg{index: idx, err: err}
	}
}

// --- Update ---
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// 通用退出
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}

		// 根据状态处理按键
		if m.state == stateConfigDir || m.state == stateConfigFF {
			if msg.Type == tea.KeyEnter {
				val := strings.TrimSpace(m.textInput.Value())
				if m.state == stateConfigDir {
					// 1. 保存 B站路径 -> 转到 FFmpeg
					m.cfg.BiliDir = val
					m.state = stateConfigFF
					m.textInput.Reset()
					m.textInput.Placeholder = "请输入 FFmpeg 完整路径 (若已配置环境变量可留空)"
					return m, nil
				} else {
					// 2. 保存 FFmpeg -> 保存文件 -> 转到扫描
					m.cfg.FFmpegPath = val
					config.Save(m.cfg)
					m.state = stateScanning
					return m, tea.Batch(m.spinner.Tick, scanCmd(m.cfg.BiliDir))
				}
			}
		} else if m.state == stateList {
			// 列表交互
			if msg.String() == "enter" {
				// 开始处理选中的视频
				idx := m.table.Cursor()
				if idx >= 0 && idx < len(m.tasks) {
					m.tasks[idx].Status = "处理中..."
					m.statusMsg = fmt.Sprintf("正在处理: %s", m.tasks[idx].DisplayTitle())
					m.refreshTable()
					return m, processCmd(m.tasks[idx], m.cfg.FFmpegPath, idx)
				}
			}
		}

	case scanResultMsg:
		m.tasks = msg
		m.state = stateList
		m.refreshTable()
		m.statusMsg = fmt.Sprintf("扫描完成，共找到 %d 个视频。按 Enter 处理，按 ↑/↓ 选择。", len(m.tasks))

	case processResultMsg:
		if msg.err != nil {
			m.tasks[msg.index].Status = "❌ 失败"
			m.statusMsg = fmt.Sprintf("错误: %v", msg.err)
		} else {
			m.tasks[msg.index].Status = "✅ 完成"
			m.statusMsg = "处理成功！文件已生成在原目录。"
		}
		m.refreshTable()

	case error:
		m.err = msg
	}

	// 组件更新
	if m.state == stateConfigDir || m.state == stateConfigFF {
		m.textInput, cmd = m.textInput.Update(msg)
	} else if m.state == stateList {
		m.table, cmd = m.table.Update(msg)
	} else if m.state == stateScanning {
		m.spinner, cmd = m.spinner.Update(msg)
	}

	return m, cmd
}

func (m *model) refreshTable() {
	rows := []table.Row{}
	for _, t := range m.tasks {
		rows = append(rows, table.Row{t.Status, t.Info.Uname, t.DisplayTitle()})
	}
	m.table.SetRows(rows)
}

// --- View ---
func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("\n❌ 发生严重错误: %v\n按 Ctrl+C 退出", m.err)
	}

	header := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Render("\n  📺 Quell\n")

	switch m.state {
	case stateConfigDir:
		return header + "\n  请进行首次配置:\n\n  B站下载目录:\n  " + m.textInput.View() + "\n"
	case stateConfigFF:
		return header + "\n  FFmpeg配置 (可选):\n\n  FFmpeg可执行文件路径:\n  " + m.textInput.View() + "\n"
	case stateScanning:
		return header + "\n  " + m.spinner.View() + " 正在扫描目录，请稍候...\n"
	case stateList:
		return header + "\n" + m.table.View() + "\n\n  " + lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(m.statusMsg) + "\n"
	}
	return ""
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
