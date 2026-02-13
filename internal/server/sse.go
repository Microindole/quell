package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// EventBroker 管理 SSE 客户端连接和事件广播
type EventBroker struct {
	mu      sync.RWMutex
	clients map[chan string]struct{}
}

// NewEventBroker 创建事件代理
func NewEventBroker() *EventBroker {
	return &EventBroker{
		clients: make(map[chan string]struct{}),
	}
}

// Subscribe 注册一个 SSE 客户端
func (b *EventBroker) Subscribe() chan string {
	ch := make(chan string, 64)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

// Unsubscribe 注销一个 SSE 客户端
func (b *EventBroker) Unsubscribe(ch chan string) {
	b.mu.Lock()
	delete(b.clients, ch)
	close(ch)
	b.mu.Unlock()
}

// Send 向所有客户端广播事件
func (b *EventBroker) Send(eventType string, data interface{}) {
	jsonData, _ := json.Marshal(data)
	msg := fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, string(jsonData))
	b.mu.RLock()
	for ch := range b.clients {
		select {
		case ch <- msg:
		default: // channel 满则丢弃
		}
	}
	b.mu.RUnlock()
}

// ServeHTTP 实现 SSE 端点
func (b *EventBroker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	// 发一条连接确认事件
	fmt.Fprintf(w, "event: connected\ndata: {}\n\n")
	flusher.Flush()

	for {
		select {
		case msg := <-ch:
			fmt.Fprint(w, msg)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}
