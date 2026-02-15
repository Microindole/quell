//go:build bundled

package engine

import (
	"embed"
	"os"
	"path/filepath"
)

//go:embed ffmpeg.exe
var ffmpegEmbed embed.FS

func GetBundledFFmpegPath() string {
	tempDir := filepath.Join(os.TempDir(), "quell_ffmpeg")
	ffmpegPath := filepath.Join(tempDir, "ffmpeg.exe")

	if _, err := os.Stat(ffmpegPath); err == nil {
		return ffmpegPath
	}

	data, err := ffmpegEmbed.ReadFile("ffmpeg.exe")
	if err != nil {
		return ""
	}

	_ = os.MkdirAll(tempDir, 0755)
	if err := os.WriteFile(ffmpegPath, data, 0755); err != nil {
		return ""
	}

	return ffmpegPath
}
