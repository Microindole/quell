package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	port := ":9999"
	pid := os.Getpid()

	// 1. 设置信号监听 (为了验证 Graceful Kill)
	// 我们监听 SIGTERM (kill) 和 SIGINT (Ctrl+C)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		// 阻塞等待信号
		sig := <-c
		fmt.Printf("\n[TARGET] 🏳️  哎哟！我收到了信号: %v\n", sig)
		fmt.Println("[TARGET] 正在收拾行李准备优雅退出... (模拟耗时 1秒)")
		time.Sleep(1 * time.Second)
		fmt.Println("[TARGET] 再见！")
		os.Exit(0)
	}()

	// 2. 启动 HTTP 服务占领端口
	fmt.Printf("\n[TARGET] 🎯 靶子进程已启动 (PID: %d)\n", pid)
	fmt.Printf("[TARGET] 正在监听端口 %s，请打开 Quell 来杀我吧！\n", port)

	if err := http.ListenAndServe(port, nil); err != nil {
		fmt.Printf("启动失败: %v\n", err)
	}
}
