package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"quell/internal/domain"
)

// Scan 遍历 BiliDir 下的一级目录
func Scan(biliDir string) ([]domain.VideoTask, error) {
	var tasks []domain.VideoTask

	entries, err := os.ReadDir(biliDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		fullPath := filepath.Join(biliDir, entry.Name())
		info, ok := parseMeta(fullPath)

		// 只有包含 .m4s 或者是合法的缓存目录才加入列表
		if !ok && !hasM4S(fullPath) {
			continue
		}

		tasks = append(tasks, domain.VideoTask{
			Dir:        fullPath,
			FolderName: entry.Name(),
			Info:       info,
			Status:     "等待",
		})
	}
	return tasks, nil
}

// parseMeta 尝试读取 videoInfo.json 或 .videoInfo
func parseMeta(dir string) (domain.BiliVideoInfo, bool) {
	var info domain.BiliVideoInfo
	// B站缓存可能有两种元数据文件名
	filenames := []string{"videoInfo.json", ".videoInfo"}

	for _, fname := range filenames {
		path := filepath.Join(dir, fname)
		data, err := os.ReadFile(path)
		if err == nil {
			if json.Unmarshal(data, &info) == nil {
				return info, true
			}
		}
	}
	return info, false
}

func hasM4S(dir string) bool {
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".m4s" {
			return true
		}
	}
	return false
}
