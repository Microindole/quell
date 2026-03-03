package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"quell/internal/config"
	"quell/internal/crawler"
	"quell/internal/domain"
	"quell/internal/downloader"
	"quell/internal/engine"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App 是 Wails 应用的后端结构体，暴露方法给前端调用
type App struct {
	ctx   context.Context
	cfg   config.Config
	tasks []domain.VideoTask
}

type LoginStartResult struct {
	URL       string `json:"url"`
	QrcodeKey string `json:"qrcode_key"`
}

type LoginPollResult struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (a *App) remoteOutputDir() string {
	if a.cfg.OutputDir != "" {
		return a.cfg.OutputDir
	}
	return a.cfg.BiliDir
}

// NewApp 创建 App 实例
func NewApp() *App {
	cfg, err := config.Load()
	if err != nil || cfg == nil {
		cfg = &config.Config{}
	}
	return &App{cfg: *cfg}
}

// startup 在 Wails 应用启动时调用
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// 如果有嵌入的 FFmpeg 且当前配置为空，自动设置
	if bundledPath := engine.GetBundledFFmpegPath(); bundledPath != "" {
		if a.cfg.FFmpegPath == "" {
			a.cfg.FFmpegPath = bundledPath
		}
	}

	if a.cfg.SESSDATA != "" {
		crawler.SetSessdata(a.cfg.SESSDATA)
		downloader.SetSessdata(a.cfg.SESSDATA)
	}
}

// --- 配置 ---

func (a *App) GetConfig() config.Config {
	return a.cfg
}

func (a *App) SaveConfig(cfg config.Config) error {
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}
	a.cfg = cfg
	if cfg.SESSDATA != "" {
		crawler.SetSessdata(cfg.SESSDATA)
		downloader.SetSessdata(cfg.SESSDATA)
	}
	return nil
}

// --- 登录 ---

func (a *App) StartBiliLogin() (*LoginStartResult, error) {
	loginURL, qrcodeKey, err := crawler.GenerateWebLoginQRCode()
	if err != nil {
		return nil, err
	}
	return &LoginStartResult{
		URL:       loginURL,
		QrcodeKey: qrcodeKey,
	}, nil
}

func (a *App) PollBiliLogin(qrcodeKey string) (*LoginPollResult, error) {
	status, message, sess, err := crawler.PollWebLoginStatus(qrcodeKey)
	if err != nil {
		return nil, err
	}
	if status == "success" && sess != "" {
		a.cfg.SESSDATA = sess
		if err := config.Save(a.cfg); err != nil {
			return nil, fmt.Errorf("登录成功但保存配置失败: %w", err)
		}
		crawler.SetSessdata(sess)
		downloader.SetSessdata(sess)
	}
	return &LoginPollResult{
		Status:  status,
		Message: message,
	}, nil
}

// --- 本地扫描 ---

func (a *App) ScanVideos() ([]domain.VideoTask, error) {
	if a.cfg.BiliDir == "" {
		return nil, fmt.Errorf("未配置 B 站目录")
	}
	tasks, err := engine.Scan(a.cfg.BiliDir, a.cfg.OutputDir, a.cfg.OutputFormat)
	if err != nil {
		return nil, fmt.Errorf("扫描失败: %w", err)
	}
	outputOnlyTasks, err := engine.ScanOutputOnlyVideos(a.remoteOutputDir())
	if err != nil {
		return nil, fmt.Errorf("扫描输出目录视频失败: %w", err)
	}

	if len(outputOnlyTasks) > 0 {
		exists := make(map[string]struct{}, len(tasks))
		for _, t := range tasks {
			if t.OutputPath != "" {
				exists[t.OutputPath] = struct{}{}
			}
		}
		for _, ot := range outputOnlyTasks {
			if _, ok := exists[ot.OutputPath]; ok {
				continue
			}
			tasks = append(tasks, ot)
		}
	}
	// WebView2 无法访问本地 file:// 路径，转换为 base64 data URL
	for i := range tasks {
		if dataURL := coverDataURL(tasks[i].Dir); dataURL != "" {
			tasks[i].Info.CoverPath = dataURL
		}
	}
	a.tasks = tasks
	return tasks, nil
}

// coverDataURL 读取封面图片并返回 base64 data URL，WebView2 可直接用于 <img src="">
func coverDataURL(dir string) string {
	for _, name := range []string{"image.jpg", "group.jpg"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err == nil {
			return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(data)
		}
	}
	return ""
}

// --- 本地合并 ---

func (a *App) MergeVideo(index int) {
	if index < 0 || index >= len(a.tasks) {
		runtime.EventsEmit(a.ctx, "merge", map[string]interface{}{
			"index": index, "status": "error", "error": "无效的索引",
		})
		return
	}

	task := a.tasks[index]

	go func() {
		runtime.EventsEmit(a.ctx, "merge", map[string]interface{}{
			"index": index, "status": "processing",
			"title": task.DisplayTitle(),
		})

		outPath, err := engine.RunMerge(task, engine.MergeConfig{
			FFmpegPath:   a.cfg.FFmpegPath,
			OutputFormat: a.cfg.OutputFormat,
			OutputDir:    a.cfg.OutputDir,
			OnProgress: func(msg string) {
				runtime.EventsEmit(a.ctx, "merge_progress", map[string]interface{}{
					"index": index, "message": msg,
				})
			},
		})
		if err != nil {
			runtime.EventsEmit(a.ctx, "merge", map[string]interface{}{
				"index": index, "status": "error", "error": err.Error(),
			})
		} else {
			runtime.EventsEmit(a.ctx, "merge", map[string]interface{}{
				"index": index, "status": "done", "output": outPath,
			})
		}
	}()
}

func (a *App) BatchMergeVideo() {
	go func() {
		runtime.EventsEmit(a.ctx, "batch_merge", map[string]interface{}{"status": "started"})
		engine.BatchMerge(a.tasks, engine.MergeConfig{
			FFmpegPath:   a.cfg.FFmpegPath,
			OutputFormat: a.cfg.OutputFormat,
			OutputDir:    a.cfg.OutputDir,
			OnProgress: func(msg string) {
				runtime.EventsEmit(a.ctx, "batch_merge_progress", map[string]interface{}{"message": msg})
			},
		})
		runtime.EventsEmit(a.ctx, "batch_merge", map[string]interface{}{"status": "done"})
		// 重新扫描以更新状态
		if _, err := a.ScanVideos(); err != nil {
			runtime.EventsEmit(a.ctx, "batch_merge", map[string]interface{}{
				"status": "rescan_error",
				"error":  err.Error(),
			})
		}
	}()
}

// --- 远程：搜索用户 ---

type SearchResult struct {
	Type  string                 `json:"type"`
	UID   string                 `json:"uid,omitempty"`
	Users []crawler.BiliUserMeta `json:"users,omitempty"`
}

func (a *App) SearchUser(keyword string) (*SearchResult, error) {
	if regexp.MustCompile(`^\d+$`).MatchString(keyword) {
		return &SearchResult{Type: "uid", UID: keyword}, nil
	}

	users, err := crawler.SearchUsers(keyword)
	if err != nil {
		return nil, fmt.Errorf("搜索用户失败: %w", err)
	}
	return &SearchResult{Type: "users", Users: users}, nil
}

// --- 远程：获取视频列表 ---

type VideoListResult struct {
	Videos     []crawler.BiliVideoMeta `json:"videos"`
	Total      int                     `json:"total"`
	Page       int                     `json:"page"`
	PageSize   int                     `json:"page_size"`
	TotalPages int                     `json:"total_pages"`
}

func (a *App) GetUserVideos(uid string, page int, pageSize int) (*VideoListResult, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 30
	}
	videos, total, err := crawler.GetUserVideos(uid, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("获取视频列表失败: %w", err)
	}
	totalPages := 0
	if total > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}
	return &VideoListResult{
		Videos:     videos,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// --- 远程：获取分P列表 ---

// VideoPageResult 分P查询结果
type VideoPageResult struct {
	Title         string                  `json:"title"`
	Pages         []downloader.PageInfo   `json:"pages"`
	StreamOptions []downloader.StreamMeta `json:"stream_options"`
}

// GetVideoPages 获取视频分P列表，用于下载前的分P选择
func (a *App) GetVideoPages(bvid string) (*VideoPageResult, error) {
	info, err := downloader.GetVideoInfo(bvid)
	if err != nil {
		return nil, fmt.Errorf("获取视频信息失败: %w", err)
	}

	streamOptions, err := downloader.ListVideoStreamMetas(info.Aid, info.Cid)
	if err != nil {
		return nil, fmt.Errorf("获取可选画质失败: %w", err)
	}
	return &VideoPageResult{
		Title:         info.Title,
		Pages:         info.Pages,
		StreamOptions: streamOptions,
	}, nil
}

// --- 远程：下载视频 ---

// DownloadVideoPages 下载用户选择的指定分P
func (a *App) DownloadVideoPages(bvid string, pages []downloader.PageInfo, title string, pref downloader.DownloadPreference) {
	go func() {
		lastMsg := ""
		outputDir := a.remoteOutputDir()
		defer func() {
			if r := recover(); r != nil {
				runtime.EventsEmit(a.ctx, "download", map[string]interface{}{
					"bvid":         bvid,
					"status":       "error",
					"error":        fmt.Sprintf("下载任务异常崩溃: %v", r),
					"output_dir":   outputDir,
					"last_message": lastMsg,
				})
			}
		}()
		runtime.EventsEmit(a.ctx, "download", map[string]interface{}{
			"bvid": bvid, "title": title, "status": "started", "output_dir": outputDir,
		})
		err := downloader.DownloadPages(bvid, pages, outputDir, a.cfg.FFmpegPath, pref, func(msg string) {
			lastMsg = msg
			runtime.EventsEmit(a.ctx, "progress", map[string]interface{}{
				"bvid": bvid, "message": msg,
			})
		})
		if err != nil {
			runtime.EventsEmit(a.ctx, "download", map[string]interface{}{
				"bvid": bvid, "status": "error", "error": err.Error(), "output_dir": outputDir, "last_message": lastMsg,
			})
		} else {
			runtime.EventsEmit(a.ctx, "download", map[string]interface{}{
				"bvid": bvid, "status": "done", "output_dir": outputDir, "last_message": lastMsg,
			})
		}
	}()
}

func (a *App) DownloadVideo(bvid string, title string, length string, pref downloader.DownloadPreference) {
	go func() {
		lastMsg := ""
		outputDir := a.remoteOutputDir()
		expectedDurationSec := int64(0)
		if sec, err := parseClockDurationToSeconds(length); err == nil {
			expectedDurationSec = sec
		}
		defer func() {
			if r := recover(); r != nil {
				runtime.EventsEmit(a.ctx, "download", map[string]interface{}{
					"bvid":         bvid,
					"status":       "error",
					"error":        fmt.Sprintf("下载任务异常崩溃: %v", r),
					"output_dir":   outputDir,
					"last_message": lastMsg,
				})
			}
		}()
		runtime.EventsEmit(a.ctx, "download", map[string]interface{}{
			"bvid": bvid, "title": title, "status": "started", "output_dir": outputDir,
		})
		err := downloader.DownloadVideo(bvid, outputDir, a.cfg.FFmpegPath, expectedDurationSec, pref, func(msg string) {
			lastMsg = msg
			runtime.EventsEmit(a.ctx, "progress", map[string]interface{}{
				"bvid": bvid, "message": msg,
			})
		})
		if err != nil {
			runtime.EventsEmit(a.ctx, "download", map[string]interface{}{
				"bvid": bvid, "status": "error", "error": err.Error(), "output_dir": outputDir, "last_message": lastMsg,
			})
		} else {
			runtime.EventsEmit(a.ctx, "download", map[string]interface{}{
				"bvid": bvid, "status": "done", "output_dir": outputDir, "last_message": lastMsg,
			})
		}
	}()
}

func parseClockDurationToSeconds(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}
	parts := strings.Split(s, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return 0, fmt.Errorf("invalid duration format: %s", s)
	}
	total := int64(0)
	for _, p := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return 0, fmt.Errorf("invalid duration token: %s", p)
		}
		total = total*60 + int64(n)
	}
	return total, nil
}

// --- 对话框 ---

// OpenDirectoryDialog 打开原生目录选择对话框
func (a *App) OpenDirectoryDialog() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择 B 站下载目录",
	})
}

func (a *App) SelectOutputDir() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择视频导出目录",
	})
}

func (a *App) OpenOutputDir() error {
	dir := a.cfg.OutputDir
	if dir == "" {
		return fmt.Errorf("未配置导出目录")
	}
	return a.OpenFolder(dir)
}

// OpenFileDialog 打开原生文件选择对话框（用于选 FFmpeg）
func (a *App) OpenFileDialog() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择 FFmpeg 可执行文件",
		Filters: []runtime.FileFilter{
			{DisplayName: "可执行文件 (*.exe)", Pattern: "*.exe"},
			{DisplayName: "所有文件", Pattern: "*"},
		},
	})
}

// --- 工具 ---

func (a *App) OpenFolder(path string) error {
	cmd := exec.Command("explorer", path)
	return cmd.Start()
}

func (a *App) OpenFile(path string) error {
	// 在 Windows 上，可以通过 explorer /select,path 或者直接 cmd /c start
	// 这里使用 cmd /c start 以使用默认程序打开
	cmd := exec.Command("cmd", "/c", "start", "", path)
	return cmd.Start()
}

func (a *App) OpenBrowserURL(rawURL string) {
	runtime.BrowserOpenURL(a.ctx, rawURL)
}

// --- 窗口控制 ---

func (a *App) WindowMinimise() {
	runtime.WindowMinimise(a.ctx)
}

func (a *App) WindowMaximise() {
	if runtime.WindowIsMaximised(a.ctx) {
		runtime.WindowUnmaximise(a.ctx)
	} else {
		runtime.WindowMaximise(a.ctx)
	}
}

func (a *App) WindowClose() {
	runtime.Quit(a.ctx)
}
