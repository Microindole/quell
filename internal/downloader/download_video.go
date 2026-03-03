package downloader

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// DownloadPages 下载指定的分P列表（用于用户选择部分分P下载）。
func DownloadPages(bvid string, pages []PageInfo, outputDir, ffmpegPath string, pref DownloadPreference, onStatus func(msg string)) error {
	if onStatus == nil {
		onStatus = func(string) {}
	}

	onStatus("正在获取视频信息...")
	info, err := GetVideoInfo(bvid)
	if err != nil {
		return fmt.Errorf("获取视频信息失败: %w", err)
	}
	onStatus(fmt.Sprintf("视频: %s (下载 %d/%d P)", info.Title, len(pages), len(info.Pages)))

	safeTitle := regexp.MustCompile(`[\\/*?:"<>|]`).ReplaceAllString(info.Title, "_")
	multiPage := len(info.Pages) > 1
	coverPath := ""

	if info.Pic != "" {
		coverPath = filepath.Join(outputDir, safeTitle+".jpg")
		onStatus("正在下载封面...")
		if err := DownloadFile(info.Pic, coverPath, func(ProgressInfo) {}); err != nil {
			onStatus("封面下载失败，将继续下载视频")
			coverPath = ""
		}
	}

	for _, page := range pages {
		var filePrefix string
		if multiPage {
			safePart := sanitizeFilename(page.Part)
			filePrefix = fmt.Sprintf("%s_P%02d_%s", safeTitle, page.Page, safePart)
		} else {
			filePrefix = safeTitle
		}

		if _, err := downloadSinglePage(info, page, 0, outputDir, ffmpegPath, filePrefix, coverPath, pref, onStatus); err != nil {
			return err
		}
		onStatus(fmt.Sprintf("P%d 下载完成: %s.mp4", page.Page, filePrefix))
	}

	onStatus(fmt.Sprintf("下载完成，共 %d P", len(pages)))
	if coverPath != "" {
		_ = os.Remove(coverPath)
	}
	return nil
}

// DownloadVideo 完整的视频下载流程：获取信息 -> 下载所有分P -> 合并。
func DownloadVideo(bvid, outputDir, ffmpegPath string, expectedDurationSec int64, pref DownloadPreference, onStatus func(msg string)) error {
	if onStatus == nil {
		onStatus = func(string) {}
	}

	onStatus("正在获取视频信息...")
	info, err := GetVideoInfo(bvid)
	if err != nil {
		return fmt.Errorf("获取视频信息失败: %w", err)
	}
	onStatus(fmt.Sprintf("视频: %s (UP主: %s, 共 %d P)", info.Title, info.Owner, len(info.Pages)))

	safeTitle := regexp.MustCompile(`[\\/*?:"<>|]`).ReplaceAllString(info.Title, "_")
	coverPath := ""
	if info.Pic != "" {
		coverPath = filepath.Join(outputDir, safeTitle+".jpg")
		onStatus("正在下载封面...")
		if err := DownloadFile(info.Pic, coverPath, func(ProgressInfo) {}); err != nil {
			onStatus("封面下载失败，将继续下载视频")
			coverPath = ""
		}
	}

	multiPage := len(info.Pages) > 1
	for _, page := range info.Pages {
		var filePrefix string
		if multiPage {
			safePart := sanitizeFilename(page.Part)
			filePrefix = fmt.Sprintf("%s_P%02d_%s", safeTitle, page.Page, safePart)
		} else {
			filePrefix = safeTitle
		}

		pageExpectedDurationSec := page.Duration
		if len(info.Pages) == 1 && expectedDurationSec > 0 {
			pageExpectedDurationSec = expectedDurationSec
		}
		if _, err := downloadSinglePage(info, page, pageExpectedDurationSec, outputDir, ffmpegPath, filePrefix, coverPath, pref, onStatus); err != nil {
			return err
		}
		onStatus(fmt.Sprintf("P%d 下载完成: %s.mp4", page.Page, filePrefix))
	}

	if multiPage {
		onStatus(fmt.Sprintf("全部 %d P 下载完成!", len(info.Pages)))
	} else {
		onStatus(fmt.Sprintf("下载完成: %s", filepath.Join(outputDir, safeTitle+".mp4")))
	}
	if coverPath != "" {
		_ = os.Remove(coverPath)
	}
	return nil
}
