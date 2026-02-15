package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

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

// --- 本地扫描 ---

func (a *App) ScanVideos() ([]domain.VideoTask, error) {
	if a.cfg.BiliDir == "" {
		return nil, fmt.Errorf("未配置 B 站目录")
	}
	tasks, err := engine.Scan(a.cfg.BiliDir)
	if err != nil {
		return nil, fmt.Errorf("扫描失败: %w", err)
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
	Videos []crawler.BiliVideoMeta `json:"videos"`
	Total  int                     `json:"total"`
}

func (a *App) GetUserVideos(uid string, page int) (*VideoListResult, error) {
	if page <= 0 {
		page = 1
	}
	videos, total, err := crawler.GetUserVideos(uid, page)
	if err != nil {
		return nil, fmt.Errorf("获取视频列表失败: %w", err)
	}
	return &VideoListResult{Videos: videos, Total: total}, nil
}

// --- 远程：获取分P列表 ---

// VideoPageResult 分P查询结果
type VideoPageResult struct {
	Title string                `json:"title"`
	Pages []downloader.PageInfo `json:"pages"`
}

// GetVideoPages 获取视频分P列表，用于下载前的分P选择
func (a *App) GetVideoPages(bvid string) (*VideoPageResult, error) {
	info, err := downloader.GetVideoInfo(bvid)
	if err != nil {
		return nil, fmt.Errorf("获取视频信息失败: %w", err)
	}
	return &VideoPageResult{
		Title: info.Title,
		Pages: info.Pages,
	}, nil
}

// --- 远程：下载视频 ---

// DownloadVideoPages 下载用户选择的指定分P
func (a *App) DownloadVideoPages(bvid string, pages []downloader.PageInfo, title string) {
	go func() {
		runtime.EventsEmit(a.ctx, "download", map[string]interface{}{
			"bvid": bvid, "title": title, "status": "started",
		})
		err := downloader.DownloadPages(bvid, pages, a.cfg.BiliDir, a.cfg.FFmpegPath, func(msg string) {
			runtime.EventsEmit(a.ctx, "progress", map[string]interface{}{
				"bvid": bvid, "message": msg,
			})
		})
		if err != nil {
			runtime.EventsEmit(a.ctx, "download", map[string]interface{}{
				"bvid": bvid, "status": "error", "error": err.Error(),
			})
		} else {
			runtime.EventsEmit(a.ctx, "download", map[string]interface{}{
				"bvid": bvid, "status": "done",
			})
		}
	}()
}

func (a *App) DownloadVideo(bvid string, title string) {
	go func() {
		runtime.EventsEmit(a.ctx, "download", map[string]interface{}{
			"bvid": bvid, "title": title, "status": "started",
		})
		err := downloader.DownloadVideo(bvid, a.cfg.BiliDir, a.cfg.FFmpegPath, func(msg string) {
			runtime.EventsEmit(a.ctx, "progress", map[string]interface{}{
				"bvid": bvid, "message": msg,
			})
		})
		if err != nil {
			runtime.EventsEmit(a.ctx, "download", map[string]interface{}{
				"bvid": bvid, "status": "error", "error": err.Error(),
			})
		} else {
			runtime.EventsEmit(a.ctx, "download", map[string]interface{}{
				"bvid": bvid, "status": "done",
			})
		}
	}()
}

// --- 对话框 ---

// OpenDirectoryDialog 打开原生目录选择对话框
func (a *App) OpenDirectoryDialog() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择 B 站下载目录",
	})
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
