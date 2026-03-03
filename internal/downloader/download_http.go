package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	parallelWorkers  = 4
	minParallelBytes = 2 * 1024 * 1024
)

// DownloadFile 下载文件，大文件自动使用多线程分段下载，支持断点续传。
func DownloadFile(url, savePath string, onProgress func(ProgressInfo)) error {
	totalSize, rangeOK := probeDownload(url)
	if rangeOK && totalSize >= minParallelBytes {
		return downloadParallel(url, savePath, totalSize, parallelWorkers, onProgress)
	}
	return downloadSingle(url, savePath, onProgress)
}

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

func downloadParallel(url, savePath string, totalSize int64, workers int, onProgress func(ProgressInfo)) error {
	if err := os.MkdirAll(filepath.Dir(savePath), 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	state, err := LoadState(savePath)
	if err != nil || state.TotalSize != totalSize {
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

	var dlBytes int64
	for _, c := range state.Chunks {
		dlBytes += c.Downloaded
	}

	startTime := time.Now()
	initialDlBytes := dlBytes

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
		_ = state.Save(savePath)
		return err
	}
	_ = os.Remove(savePath + ".qdl")
	return nil
}

func downloadChunk(ctx context.Context, url, savePath string, state *DownloadState, idx int, totalDownloaded *int64, initialDlBytes int64, startTime time.Time, onProgress func(ProgressInfo)) error {
	c := &state.Chunks[idx]
	if c.Downloaded >= (c.End - c.Start + 1) {
		return nil
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
