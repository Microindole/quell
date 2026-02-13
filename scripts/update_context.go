package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	rootDir, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting working directory:", err)
		return
	}

	contextPath := filepath.Join(rootDir, "prompt", "context.md")
	content, err := os.ReadFile(contextPath)
	if err != nil {
		fmt.Printf("Error reading context.md: %v\n", err)
		return
	}

	newContent := string(content)

	// 更新时间戳
	updateTime := fmt.Sprintf("\n\n*Ctx Updated: %s*", time.Now().Format("2006-01-02 15:04:05"))

	if idx := strings.LastIndex(newContent, "\n\n*Ctx Updated:"); idx != -1 {
		newContent = newContent[:idx]
	}

	newContent += updateTime

	if err := os.WriteFile(contextPath, []byte(newContent), 0644); err != nil {
		fmt.Printf("Error writing context.md: %v\n", err)
		return
	}

	fmt.Println("Context updated successfully at", contextPath)
}
