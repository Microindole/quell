package engine

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"quell/internal/domain"
)

// RunMerge 执行 PowerShell 合并脚本，将 B 站缓存的 m4s 文件合并为 mp4
func RunMerge(task domain.VideoTask, ffmpegPath string, scriptFS embed.FS) (string, error) {
	scriptName := "merge_v3.ps1"
	tmpScript := filepath.Join(os.TempDir(), "quell_"+scriptName)
	data, _ := scriptFS.ReadFile("scripts/merge.ps1")
	os.WriteFile(tmpScript, data, 0755)

	safeTitle := regexp.MustCompile(`[\\/*?:"<>|]`).ReplaceAllString(task.DisplayTitle(), "_")
	outputFile := filepath.Join(task.Dir, safeTitle+".mp4")

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-File", tmpScript,
		"-TargetDir", task.Dir,
		"-OutputName", safeTitle,
		"-FFmpegPath", ffmpegPath,
		"-CoverUrl", task.Info.CoverUrl,
		"-LocalCoverPath", task.Info.CoverPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%v | Output: %s", err, string(output))
	}
	if !strings.Contains(string(output), "SUCCESS") {
		return "", fmt.Errorf("Script executed but no success signal. Output: %s", string(output))
	}
	return outputFile, nil
}
