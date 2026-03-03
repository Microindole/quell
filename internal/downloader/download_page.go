package downloader

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func downloadSinglePage(info *VideoInfo, page PageInfo, expectedDurationSec int64, outputDir, ffmpegPath, filePrefix, coverPath string, pref DownloadPreference, onStatus func(msg string)) (string, error) {
	onStatus(fmt.Sprintf("正在获取 P%d 播放地址...", page.Page))
	if expectedDurationSec <= 0 {
		expectedDurationSec = page.Duration
	}
	selection, err := GetPlayURL(info.Aid, page.Cid, expectedDurationSec, pref)
	if err != nil {
		return "", fmt.Errorf("获取 P%d 播放地址失败: %w", page.Page, err)
	}
	videoStream := selection.Video
	audioStream := selection.Audio
	onStatus(fmt.Sprintf("P%d 流信息: %s", page.Page, selection.Debug))

	videoTmp := filepath.Join(outputDir, filePrefix+"_video.m4s")
	audioTmp := filepath.Join(outputDir, filePrefix+"_audio.m4s")
	finalOutput := filepath.Join(outputDir, filePrefix+".mp4")

	formatProgress := func(prefix string, p ProgressInfo) {
		speedStr := formatSpeed(p.Speed)
		onStatus(fmt.Sprintf("P%d %s: %.1f%% (%s/%s) %s",
			page.Page, prefix, p.Percentage,
			formatBytes(p.Downloaded), formatBytes(p.Total),
			speedStr))
	}

	onStatus(fmt.Sprintf("正在下载 P%d 视频流...", page.Page))
	if err := DownloadFile(videoStream.BaseURL, videoTmp, func(p ProgressInfo) {
		formatProgress("视频", p)
	}); err != nil {
		return "", fmt.Errorf("下载 P%d 视频流失败: %w", page.Page, err)
	}

	if audioStream == nil {
		if err := os.Rename(videoTmp, finalOutput); err == nil {
			if err := attachCoverToFile(finalOutput, coverPath, ffmpegPath); err != nil {
				onStatus(fmt.Sprintf("P%d 封面写入失败，保留无封面文件: %v", page.Page, err))
			}
			onStatus(fmt.Sprintf("P%d 直链下载完成（无分离音轨）", page.Page))
			return finalOutput, nil
		}
		if err := DownloadFile(videoStream.BaseURL, finalOutput, func(p ProgressInfo) {
			formatProgress("直链", p)
		}); err != nil {
			return "", fmt.Errorf("下载 P%d 直链视频失败: %w", page.Page, err)
		}
		_ = os.Remove(videoTmp)
		if err := attachCoverToFile(finalOutput, coverPath, ffmpegPath); err != nil {
			onStatus(fmt.Sprintf("P%d 封面写入失败，保留无封面文件: %v", page.Page, err))
		}
		onStatus(fmt.Sprintf("P%d 直链下载完成（无分离音轨）", page.Page))
		return finalOutput, nil
	}

	onStatus(fmt.Sprintf("正在下载 P%d 音频流...", page.Page))
	if err := DownloadFile(audioStream.BaseURL, audioTmp, func(p ProgressInfo) {
		formatProgress("音频", p)
	}); err != nil {
		_ = os.Remove(videoTmp)
		return "", fmt.Errorf("下载 P%d 音频流失败: %w", page.Page, err)
	}

	onStatus(fmt.Sprintf("正在合并 P%d 音视频...", page.Page))
	ffmpegCmd := "ffmpeg"
	if ffmpegPath != "" {
		ffmpegCmd = ffmpegPath
	}

	var output []byte
	var mergeErr error
	if coverPath != "" {
		onStatus(fmt.Sprintf("P%d 正在写入封面...", page.Page))
		cmdWithCover := exec.Command(ffmpegCmd,
			"-i", videoTmp,
			"-i", audioTmp,
			"-i", coverPath,
			"-map", "0:v:0",
			"-map", "1:a:0",
			"-map", "2:v:0",
			"-c", "copy",
			"-c:v:1", "mjpeg",
			"-disposition:v:1", "attached_pic",
			"-metadata:s:v:1", "title=Cover",
			"-metadata:s:v:1", "comment=Cover (front)",
			"-y",
			"-loglevel", "error",
			finalOutput,
		)
		output, mergeErr = cmdWithCover.CombinedOutput()
	}
	if mergeErr != nil || coverPath == "" {
		cmd := exec.Command(ffmpegCmd,
			"-i", videoTmp,
			"-i", audioTmp,
			"-c", "copy",
			"-y",
			"-loglevel", "error",
			finalOutput,
		)
		output, mergeErr = cmd.CombinedOutput()
	}
	_ = os.Remove(videoTmp)
	_ = os.Remove(audioTmp)

	if mergeErr != nil {
		return "", fmt.Errorf("p%d ffmpeg 合并失败: %w | %s", page.Page, mergeErr, string(output))
	}
	return finalOutput, nil
}

func attachCoverToFile(videoPath, coverPath, ffmpegPath string) error {
	if coverPath == "" {
		return nil
	}
	ffmpegCmd := "ffmpeg"
	if ffmpegPath != "" {
		ffmpegCmd = ffmpegPath
	}
	tmpOutput := videoPath + ".cover.mp4"
	cmd := exec.Command(ffmpegCmd,
		"-i", videoPath,
		"-i", coverPath,
		"-map", "0",
		"-map", "1:v:0",
		"-c", "copy",
		"-c:v:1", "mjpeg",
		"-disposition:v:1", "attached_pic",
		"-metadata:s:v:1", "title=Cover",
		"-metadata:s:v:1", "comment=Cover (front)",
		"-y",
		"-loglevel", "error",
		tmpOutput,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		_ = os.Remove(tmpOutput)
		return fmt.Errorf("ffmpeg 写封面失败: %w | %s", err, string(output))
	}
	if err := os.Rename(tmpOutput, videoPath); err != nil {
		return fmt.Errorf("替换封面文件失败: %w", err)
	}
	return nil
}
