package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// --- 3. HTTP 文件下载 ---

const (
	parallelWorkers  = 4
	minParallelBytes = 2 * 1024 * 1024 // 2MB 以上才启用多线程
)

// DownloadFile 下载文件，大文件自动使用多线程分段下载，支持断点续传
func DownloadFile(url, savePath string, onProgress func(ProgressInfo)) error {
	totalSize, rangeOK := probeDownload(url)
	if rangeOK && totalSize >= minParallelBytes {
		return downloadParallel(url, savePath, totalSize, parallelWorkers, onProgress)
	}
	return downloadSingle(url, savePath, onProgress)
}

// probeDownload 发送 HEAD 请求探测文件大小及是否支持 Range 下载
func probeDownload(url string) (int64, bool) {
	req, _ := http.NewRequest("HEAD", url, nil)
	addCommonHeaders(req)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return 0, false
	}
	defer resp.Body.Close()
	supportsRange := strings.EqualFold(resp.Header.Get("Accept-Ranges"), "bytes")
	return resp.ContentLength, supportsRange && resp.ContentLength > 0
}

// downloadParallel 多线程分段下载
func downloadParallel(url, savePath string, totalSize int64, workers int, onProgress func(ProgressInfo)) error {
	if err := os.MkdirAll(filepath.Dir(savePath), 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	state, err := LoadState(savePath)
	if err != nil || state.TotalSize != totalSize {
		// 初始化新下载
		state = &DownloadState{
			TotalSize: totalSize,
			Chunks:    make([]ChunkState, workers),
		}
		chunkSize := totalSize / int64(workers)
		for i := 0; i < workers; i++ {
			state.Chunks[i].Start = int64(i) * chunkSize
			if i == workers-1 {
				state.Chunks[i].End = totalSize - 1
			} else {
				state.Chunks[i].End = state.Chunks[i].Start + chunkSize - 1
			}
		}

		// 创建并预分配文件
		f, err := os.Create(savePath)
		if err != nil {
			return fmt.Errorf("创建文件失败: %w", err)
		}
		if err := f.Truncate(totalSize); err != nil {
			f.Close()
			return fmt.Errorf("预分配文件失败: %w", err)
		}
		f.Close()
	}

	// 累计已下载字节
	var dlBytes int64
	for _, c := range state.Chunks {
		dlBytes += c.Downloaded
	}

	// 记录开始时间用于计算速度
	startTime := time.Now()
	initialDlBytes := dlBytes

	// 并发下载各分段
	var wg sync.WaitGroup
	errCh := make(chan error, workers)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for i := range state.Chunks {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			if err := downloadChunk(ctx, url, savePath, state, idx, &dlBytes, initialDlBytes, startTime, onProgress); err != nil {
				if ctx.Err() == nil {
					select {
					case errCh <- err:
					default:
					}
					cancel()
				}
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	if err := <-errCh; err != nil {
		// 出错时保存状态
		_ = state.Save(savePath)
		return err
	}

	// 下载完成后删除状态文件
	_ = os.Remove(savePath + ".qdl")
	return nil
}

// downloadChunk 下载指定字节范围并写入文件对应偏移
func downloadChunk(ctx context.Context, url, savePath string, state *DownloadState, idx int, totalDownloaded *int64, initialDlBytes int64, startTime time.Time, onProgress func(ProgressInfo)) error {
	c := &state.Chunks[idx]
	if c.Downloaded >= (c.End - c.Start + 1) {
		return nil // 该段已完成
	}

	actualStart := c.Start + c.Downloaded
	req, _ := http.NewRequest("GET", url, nil)
	addCommonHeaders(req)
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", actualStart, c.End))
	req = req.WithContext(ctx)

	client := &http.Client{Timeout: 30 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("分段请求失败 [%d-%d]: %w", actualStart, c.End, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("服务器拒绝分段请求 (HTTP %d)", resp.StatusCode)
	}

	file, err := os.OpenFile(savePath, os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	buf := make([]byte, 256*1024)
	offset := actualStart
	lastSave := time.Now()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := file.WriteAt(buf[:n], offset); writeErr != nil {
				return fmt.Errorf("写入失败: %w", writeErr)
			}
			n64 := int64(n)
			offset += n64
			c.Downloaded += n64
			total := atomic.AddInt64(totalDownloaded, n64)

			if onProgress != nil {
				elapsed := time.Since(startTime).Seconds()
				speed := 0.0
				if elapsed > 0 {
					speed = float64(total-initialDlBytes) / elapsed
				}
				onProgress(ProgressInfo{
					Downloaded: total,
					Total:      state.TotalSize,
					Percentage: float64(total) / float64(state.TotalSize) * 100,
					Speed:      speed,
				})
			}

			// 每 5 秒保存一次状态，防止频繁 I/O
			if time.Since(lastSave) > 5*time.Second {
				_ = state.Save(savePath)
				lastSave = time.Now()
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return fmt.Errorf("读取数据失败: %w", readErr)
		}
	}
	return nil
}

// downloadSingle 单线程下载（文件较小或服务器不支持 Range 时使用）
func downloadSingle(url, savePath string, onProgress func(ProgressInfo)) error {
	req, _ := http.NewRequest("GET", url, nil)
	addCommonHeaders(req)

	client := &http.Client{Timeout: 30 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("请求下载地址失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("HTTP 状态码异常: %d", resp.StatusCode)
	}

	totalSize := resp.ContentLength
	if err := os.MkdirAll(filepath.Dir(savePath), 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}
	file, err := os.Create(savePath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()

	buf := make([]byte, 256*1024)
	var downloaded int64
	startTime := time.Now()

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := file.Write(buf[:n]); writeErr != nil {
				return fmt.Errorf("写入文件失败: %w", writeErr)
			}
			downloaded += int64(n)
			if onProgress != nil {
				elapsed := time.Since(startTime).Seconds()
				speed := 0.0
				if elapsed > 0 {
					speed = float64(downloaded) / elapsed
				}
				onProgress(ProgressInfo{
					Downloaded: downloaded,
					Total:      totalSize,
					Percentage: float64(downloaded) / float64(totalSize) * 100,
					Speed:      speed,
				})
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return fmt.Errorf("读取数据失败: %w", readErr)
		}
	}
	return nil
}

// --- 4. 高层封装 ---

// downloadSinglePage 下载单个分P的音视频并合并
func downloadSinglePage(info *VideoInfo, page PageInfo, expectedDurationSec int64, outputDir, ffmpegPath, filePrefix, coverPath string, pref DownloadPreference, onStatus func(msg string)) (string, error) {
	// 获取播放地址
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

	// 准备文件路径
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

	// 下载视频流
	onStatus(fmt.Sprintf("正在下载 P%d 视频流...", page.Page))
	if err := DownloadFile(videoStream.BaseURL, videoTmp, func(p ProgressInfo) {
		formatProgress("视频", p)
	}); err != nil {
		return "", fmt.Errorf("下载 P%d 视频流失败: %w", page.Page, err)
	}

	// 兼容无分离音轨的直链视频：直接输出 mp4，不进入合并流程。
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

	// 下载音频流
	onStatus(fmt.Sprintf("正在下载 P%d 音频流...", page.Page))
	if err := DownloadFile(audioStream.BaseURL, audioTmp, func(p ProgressInfo) {
		formatProgress("音频", p)
	}); err != nil {
		os.Remove(videoTmp)
		return "", fmt.Errorf("下载 P%d 音频流失败: %w", page.Page, err)
	}

	// ffmpeg 合并
	onStatus(fmt.Sprintf("正在合并 P%d 音视频...", page.Page))
	ffmpegCmd := "ffmpeg"
	if ffmpegPath != "" {
		ffmpegCmd = ffmpegPath
	}

	var output []byte
	var mergeErr error
	if coverPath != "" {
		// 优先尝试写入封面元数据，失败后再回退到普通合并，避免任务整体失败。
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
	os.Remove(videoTmp)
	os.Remove(audioTmp)

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

// DownloadPages 下载指定的分P列表（用于用户选择部分分P下载）
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

// DownloadVideo 完整的视频下载流程：获取信息 -> 下载所有分P -> 合并
func DownloadVideo(bvid, outputDir, ffmpegPath string, expectedDurationSec int64, pref DownloadPreference, onStatus func(msg string)) error {
	if onStatus == nil {
		onStatus = func(string) {} // 空回调
	}

	// 1. 获取视频信息
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

	// 2. 遍历所有分P下载
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
