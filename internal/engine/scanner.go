package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"quell/internal/domain"
)

// Scan 递归遍历 BiliDir 下的目录，寻找包含视频元数据的文件夹
func Scan(biliDir, outputDir, format string) ([]domain.VideoTask, error) {
	if format == "" {
		format = "mp4"
	}
	var tasks []domain.VideoTask

	err := filepath.WalkDir(biliDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}

		info, ok := parseMeta(path)
		if !ok {
			// 如果没有元数据，但有 m4s 文件，也可能是一个任务（虽然信息不全）
			if !hasM4S(path) {
				return nil
			}
		}

		// 构造任务
		task := domain.VideoTask{
			Dir:        path,
			FolderName: filepath.Base(path),
			Info:       info,
			Status:     "等待",
		}

		// 检查是否已存在合并后的文件
		checkDir := outputDir
		if checkDir == "" {
			checkDir = path
		}
		// 使用统一的命名逻辑
		outName := GetOutputName(task, 0)
		outPath := filepath.Join(checkDir, outName+"."+format)
		if _, err := os.Stat(outPath); err == nil {
			task.Status = "完成"
			task.OutputPath = outPath
		}

		tasks = append(tasks, task)
		return nil
	})

	return tasks, err
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
