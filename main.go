package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"quell/internal/config"
	"quell/internal/crawler"
	"quell/internal/domain"
	"quell/internal/downloader"
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
		"-LocalCoverPath", task.Info.CoverPath,
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
	stateConfigDir     sessionState = iota // 输入 B站路径
	stateConfigFF                          // 输入 FFmpeg 路径
	stateModeSelect                        // 选择模式：本地合并 / 远程下载 [新增]
	stateScanning                          // 扫描中 (本地)
	stateList                              // 列表 (本地)
	stateInputUID                          // 输入 UID (远程) [新增]
	stateFetching                          // 获取列表中 (远程) [新增]
	stateRemoteList                        // 远程列表显示 [新增]
	stateDownloading                       // 下载中 [新增]
	stateSearchingUser                     // 搜索用户中 [新增]
	stateUserList                          // 用户列表选择 [新增]
)

// --- Model ---
type model struct {
	state     sessionState
	cfg       config.Config
	textInput textinput.Model
	table     table.Model
	spinner   spinner.Model
	tasks     []domain.VideoTask

	// Remote related
	remoteVideos []crawler.BiliVideoMeta // 爬取的列表
	userResults  []crawler.BiliUserMeta  // 搜索到的用户列表
	selectedUID  string
	page         int
	totalVideos  int
	downloadLog  string // 简单的下载日志

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
		if loadedCfg.SESSDATA != "" {
			crawler.SetSessdata(loadedCfg.SESSDATA)
			downloader.SetSessdata(loadedCfg.SESSDATA)
		}
		m.state = stateModeSelect // 配置好了，去选择模式
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
		} else if m.state == stateModeSelect {
			// 模式选择
			if msg.String() == "1" {
				m.state = stateScanning
				return m, tea.Batch(m.spinner.Tick, scanCmd(m.cfg.BiliDir))
			} else if msg.String() == "2" {
				m.state = stateInputUID
				m.textInput.Reset()
				m.textInput.Placeholder = "请输入 UP 主 UID 或 昵称"
				return m, nil
			}
		} else if m.state == stateInputUID {
			// 输入 关键词 / UID
			if msg.Type == tea.KeyEnter {
				val := strings.TrimSpace(m.textInput.Value())
				if val != "" {
					// 纯数字视为 UID
					if regexp.MustCompile(`^\d+$`).MatchString(val) {
						m.selectedUID = val
						m.page = 1
						m.state = stateFetching
						return m, tea.Batch(m.spinner.Tick, fetchVideosCmd(val, 1))
					} else {
						// 否则搜索用户
						m.state = stateSearchingUser
						return m, tea.Batch(m.spinner.Tick, searchUserCmd(val))
					}
				}
			}
		} else if m.state == stateUserList {
			// 选择用户
			if msg.String() == "enter" {
				idx := m.table.Cursor()
				if idx >= 0 && idx < len(m.userResults) {
					u := m.userResults[idx]
					m.selectedUID = fmt.Sprintf("%d", u.Mid)
					m.page = 1
					m.state = stateFetching
					return m, tea.Batch(m.spinner.Tick, fetchVideosCmd(m.selectedUID, 1))
				}
			}
		} else if m.state == stateRemoteList {
			// 远程列表操作
			if msg.String() == "enter" {
				idx := m.table.Cursor()
				if idx >= 0 && idx < len(m.remoteVideos) {
					v := m.remoteVideos[idx]
					m.statusMsg = "开始下载: " + v.Title
					return m, downloadCmd(v.Bvid, m.cfg.BiliDir, m.cfg.FFmpegPath)
				}
			}
		}

	case fetchResultMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("获取视频列表失败: %v", msg.err)
			m.state = stateInputUID
			m.textInput.Reset()
			m.textInput.Placeholder = "请输入 UP 主 UID 或 昵称"
		} else {
			m.remoteVideos = msg.videos
			m.totalVideos = msg.total
			m.state = stateRemoteList
			m.refreshRemoteTable()
			m.statusMsg = fmt.Sprintf("获取成功: 共 %d 个视频 (当前页 %d)。按 Enter 下载。", m.totalVideos, len(m.remoteVideos))
		}

	case downloadResultMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("下载失败: %v", msg.err)
		} else {
			m.statusMsg = "下载成功! 文件已保存。"
		}

	case scanResultMsg:
		m.tasks = msg
		m.state = stateList
		m.refreshTable()
		m.statusMsg = fmt.Sprintf("扫描完成，共找到 %d 个视频。按 Enter 处理，按 ↑/↓ 选择。", len(m.tasks))

	case searchUserResultMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("搜索失败: %v", msg.err)
			m.state = stateInputUID
		} else {
			m.userResults = msg.users
			m.state = stateUserList
			m.refreshUserTable()
			m.statusMsg = fmt.Sprintf("搜索到 %d 个用户，请选择。", len(m.userResults))
		}

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
	if m.state == stateConfigDir || m.state == stateConfigFF || m.state == stateInputUID {
		m.textInput, cmd = m.textInput.Update(msg)
	} else if m.state == stateList || m.state == stateRemoteList || m.state == stateUserList {
		m.table, cmd = m.table.Update(msg)
	} else if m.state == stateScanning || m.state == stateFetching || m.state == stateDownloading || m.state == stateSearchingUser {
		m.spinner, cmd = m.spinner.Update(msg)
	}

	return m, cmd
}

func (m *model) refreshTable() {
	rows := []table.Row{}
	for _, t := range m.tasks {
		rows = append(rows, table.Row{t.Status, t.Info.Uname, t.DisplayTitle()})
	}
	m.table.SetColumns([]table.Column{
		{Title: "状态", Width: 8},
		{Title: "UP主", Width: 15},
		{Title: "标题", Width: 50},
	})
	m.table.SetRows(rows)
}

func (m *model) refreshRemoteTable() {
	rows := []table.Row{}
	for _, v := range m.remoteVideos {
		rows = append(rows, table.Row{"未下载", v.Length, v.Title})
	}
	m.table.SetColumns([]table.Column{
		{Title: "状态", Width: 8},
		{Title: "时长", Width: 10},
		{Title: "标题", Width: 60},
	})
	m.table.SetRows(rows)
}

func (m *model) refreshUserTable() {
	rows := []table.Row{}
	for _, u := range m.userResults {
		rows = append(rows, table.Row{fmt.Sprintf("%d", u.Mid), u.Uname, fmt.Sprintf("%d粉", u.Fans)})
	}
	m.table.SetColumns([]table.Column{
		{Title: "UID", Width: 12},
		{Title: "昵称", Width: 20},
		{Title: "粉丝数", Width: 10},
	})
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
	case stateModeSelect:
		return header + "\n  请选择模式:\n\n  [1] 本地缓存合并 (B站官方客户端下载的视频)\n  [2] 远程批量下载 (内置下载器 极速下载)\n\n  请按 1 或 2"
	case stateInputUID:
		s := header + "\n  批量下载模式:\n\n  请输入 UP 主 UID 或 昵称关键词:\n  " + m.textInput.View() + "\n"
		if m.statusMsg != "" {
			s += "\n  " + lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(m.statusMsg) + "\n"
		}
		return s
	case stateScanning:
		return header + "\n  " + m.spinner.View() + " 正在扫描目录，请稍候...\n"
	case stateSearchingUser:
		return header + "\n  " + m.spinner.View() + " 正在搜索用户...\n"
	case stateFetching:
		return header + "\n  " + m.spinner.View() + " 正在获取视频列表 (Wbi 签名中)..."
	case stateList:
		return header + "\n  [本地缓存列表]\n" + m.table.View() + "\n\n  " + lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(m.statusMsg) + "\n"
	case stateRemoteList:
		return header + "\n  [UP主视频列表 - 按回车下载]\n" + m.table.View() + "\n\n  " + lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(m.statusMsg) + "\n"
	case stateUserList:
		return header + "\n  [搜索结果 - 请选择UP主]\n" + m.table.View() + "\n\n  " + lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(m.statusMsg) + "\n"
	}
	return ""
}

func main() {
	// Parse flags: 默认启动 GUI (wails dev 依赖此行为生成绑定)
	// 使用 -tui 参数可切换到终端模式
	tuiMode := false
	for _, arg := range os.Args {
		if arg == "-tui" {
			tuiMode = true
			break
		}
	}

	if tuiMode {
		// Start TUI
		p := tea.NewProgram(initialModel())
		if _, err := p.Run(); err != nil {
			fmt.Printf("启动 TUI 失败: %v", err)
			os.Exit(1)
		}
		return
	}

	// Start GUI (default)
	startGUI()
}
