package transfer

import (
	"context"
	"net"
	"net/http"
	"path/filepath"
	"time"
)

// GetLocalIPs 获取所有非回环的 IPv4 地址
func GetLocalIPs() ([]string, error) {
	var ips []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ips = append(ips, ipnet.IP.String())
			}
		}
	}
	return ips, nil
}

// ServeFile 启动 HTTP 服务
func ServeFile(filePath string) (int, func(), error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, nil, err
	}
	port := listener.Addr().(*net.TCPAddr).Port

	// 创建一个 Server 实例
	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			filename := filepath.Base(filePath)
			w.Header().Set("Content-Disposition", "attachment; filename="+filename)
			http.ServeFile(w, r, filePath)
		}),
	}

	// 🔥 安全机制：创建一个带超时的 Context (例如 60 分钟)
	// 防止僵尸进程无限期占用端口
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)

	go func() {
		// 启动服务
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			// 服务异常退出
		}
	}()

	// 监听超时自动关闭
	go func() {
		<-ctx.Done()
		server.Close()
	}()

	// 返回手动关闭函数
	stopFunc := func() {
		cancel()       // 取消 Context
		server.Close() // 立即关闭 Server
	}

	return port, stopFunc, nil
}
