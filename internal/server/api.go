package server

import (
	"embed"
	"encoding/json"
	"net/http"
	"regexp"
	"sync"

	"quell/internal/config"
	"quell/internal/crawler"
	"quell/internal/domain"
	"quell/internal/downloader"
	"quell/internal/engine"
)

// apiHandler 持有 API 所需的状态
type apiHandler struct {
	mu       sync.Mutex
	cfg      config.Config
	scriptFS embed.FS
	events   *EventBroker
	tasks    []domain.VideoTask // 缓存的扫描结果
}

// --- 配置 ---

func (h *apiHandler) getConfig(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	cfg := h.cfg
	h.mu.Unlock()
	writeJSON(w, cfg)
}

func (h *apiHandler) saveConfig(w http.ResponseWriter, r *http.Request) {
	var cfg config.Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		httpError(w, "无效的请求体", 400)
		return
	}
	if err := config.Save(cfg); err != nil {
		httpError(w, "保存配置失败: "+err.Error(), 500)
		return
	}
	h.mu.Lock()
	h.cfg = cfg
	h.mu.Unlock()

	// 更新 SESSDATA
	if cfg.SESSDATA != "" {
		crawler.SetSessdata(cfg.SESSDATA)
		downloader.SetSessdata(cfg.SESSDATA)
	}

	writeJSON(w, map[string]string{"status": "ok"})
}

// --- 本地扫描 ---

func (h *apiHandler) scanVideos(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	biliDir := h.cfg.BiliDir
	h.mu.Unlock()

	if biliDir == "" {
		httpError(w, "未配置 B 站目录", 400)
		return
	}

	tasks, err := engine.Scan(biliDir)
	if err != nil {
		httpError(w, "扫描失败: "+err.Error(), 500)
		return
	}

	h.mu.Lock()
	h.tasks = tasks
	h.mu.Unlock()

	writeJSON(w, tasks)
}

// --- 本地合并 ---

func (h *apiHandler) mergeVideo(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Index int `json:"index"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, "无效的请求体", 400)
		return
	}

	h.mu.Lock()
	if req.Index < 0 || req.Index >= len(h.tasks) {
		h.mu.Unlock()
		httpError(w, "无效的索引", 400)
		return
	}
	task := h.tasks[req.Index]
	ffmpegPath := h.cfg.FFmpegPath
	scriptFS := h.scriptFS
	h.mu.Unlock()

	go func() {
		h.events.Send("merge", map[string]interface{}{
			"index": req.Index, "status": "processing",
			"title": task.DisplayTitle(),
		})
		err := engine.RunMerge(task, ffmpegPath, scriptFS)
		if err != nil {
			h.events.Send("merge", map[string]interface{}{
				"index": req.Index, "status": "error", "error": err.Error(),
			})
		} else {
			h.events.Send("merge", map[string]interface{}{
				"index": req.Index, "status": "done",
			})
		}
	}()

	writeJSON(w, map[string]string{"status": "started"})
}

// --- 远程：搜索用户 ---

func (h *apiHandler) searchUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Keyword string `json:"keyword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, "无效的请求体", 400)
		return
	}

	// 纯数字视为 UID，直接返回特殊响应
	if regexp.MustCompile(`^\d+$`).MatchString(req.Keyword) {
		writeJSON(w, map[string]interface{}{
			"type": "uid",
			"uid":  req.Keyword,
		})
		return
	}

	users, err := crawler.SearchUsers(req.Keyword)
	if err != nil {
		httpError(w, "搜索用户失败: "+err.Error(), 500)
		return
	}

	writeJSON(w, map[string]interface{}{
		"type":  "users",
		"users": users,
	})
}

// --- 远程：获取视频列表 ---

func (h *apiHandler) getUserVideos(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UID  string `json:"uid"`
		Page int    `json:"page"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, "无效的请求体", 400)
		return
	}
	if req.Page <= 0 {
		req.Page = 1
	}

	videos, total, err := crawler.GetUserVideos(req.UID, req.Page)
	if err != nil {
		httpError(w, "获取视频列表失败: "+err.Error(), 500)
		return
	}

	writeJSON(w, map[string]interface{}{
		"videos": videos,
		"total":  total,
	})
}

// --- 远程：下载视频 ---

func (h *apiHandler) downloadVideo(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Bvid  string `json:"bvid"`
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, "无效的请求体", 400)
		return
	}

	h.mu.Lock()
	biliDir := h.cfg.BiliDir
	ffmpegPath := h.cfg.FFmpegPath
	h.mu.Unlock()

	go func() {
		h.events.Send("download", map[string]interface{}{
			"bvid": req.Bvid, "title": req.Title, "status": "started",
		})
		err := downloader.DownloadVideo(req.Bvid, biliDir, ffmpegPath, func(msg string) {
			h.events.Send("progress", map[string]interface{}{
				"bvid": req.Bvid, "message": msg,
			})
		})
		if err != nil {
			h.events.Send("download", map[string]interface{}{
				"bvid": req.Bvid, "status": "error", "error": err.Error(),
			})
		} else {
			h.events.Send("download", map[string]interface{}{
				"bvid": req.Bvid, "status": "done",
			})
		}
	}()

	writeJSON(w, map[string]string{"status": "started"})
}

// --- 工具函数 ---

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func httpError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
