package main

import (
	"context"
	"embed"
	"fmt"
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
	ctx      context.Context
	cfg      config.Config
	scriptFS embed.FS
	tasks    []domain.VideoTask
}

// NewApp 创建 App 实例
func NewApp(scriptFS embed.FS) *App {
	cfg, err := config.Load()
	if err != nil || cfg == nil {
		cfg = &config.Config{}
	}
	return &App{
		cfg:      *cfg,
		scriptFS: scriptFS,
	}
}

// startup 在 Wails 应用启动时调用
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// 初始化 SESSDATA
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
	a.tasks = tasks
	return tasks, nil
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

		outPath, err := engine.RunMerge(task, a.cfg.FFmpegPath, a.scriptFS)
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
	// 纯数字视为 UID
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

// --- 远程：下载视频 ---

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

// --- 工具 ---

func (a *App) OpenFolder(path string) error {
	safeTitle := regexp.MustCompile(`[\\/*?:"<>|]`).ReplaceAllString(filepath.Base(path), "_")
	_ = safeTitle
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
