package server

import (
	"embed"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os/exec"
	"runtime"

	"quell/internal/config"
)

// StartServer 启动嵌入式 Web GUI 服务
func StartServer(cfg config.Config, scriptFS embed.FS, webFS embed.FS) error {
	events := NewEventBroker()
	api := &apiHandler{
		cfg:      cfg,
		scriptFS: scriptFS,
		events:   events,
	}

	mux := http.NewServeMux()

	// API 路由
	mux.HandleFunc("GET /api/config", api.getConfig)
	mux.HandleFunc("POST /api/config", api.saveConfig)
	mux.HandleFunc("POST /api/scan", api.scanVideos)
	mux.HandleFunc("POST /api/merge", api.mergeVideo)
	mux.HandleFunc("POST /api/search/user", api.searchUser)
	mux.HandleFunc("POST /api/user/videos", api.getUserVideos)
	mux.HandleFunc("POST /api/download", api.downloadVideo)
	mux.Handle("GET /api/events", events)

	// 静态文件（嵌入的 web/ 目录）
	webContent, err := fs.Sub(webFS, "web")
	if err != nil {
		return fmt.Errorf("无法加载嵌入的 Web 资源: %w", err)
	}
	mux.Handle("/", http.FileServer(http.FS(webContent)))

	// 监听随机端口
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("无法启动服务器: %w", err)
	}

	addr := listener.Addr().String()
	url := "http://" + addr
	fmt.Printf("Quell GUI 已启动: %s\n", url)
	fmt.Println("在浏览器中打开上方地址，按 Ctrl+C 关闭")

	go openBrowser(url)

	return http.Serve(listener, mux)
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Run()
}
