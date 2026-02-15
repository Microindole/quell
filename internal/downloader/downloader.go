package downloader

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"quell/internal/crawler"
)

// --- Cookie 支持 ---

var sessdata string

// SetSessdata 设置 SESSDATA Cookie，用于解锁高清画质
func SetSessdata(s string) {
	sessdata = s
}

// addCommonHeaders 为请求添加通用 Headers（含 Cookie）
func addCommonHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://www.bilibili.com")
	if sessdata != "" {
		req.Header.Set("Cookie", "SESSDATA="+sessdata)
	}
}

// --- 数据结构 ---

// PageInfo 单个分P信息
type PageInfo struct {
	Cid  int64  `json:"cid"`
	Part string `json:"part"`
	Page int    `json:"page"`
}

// VideoInfo 视频基本信息
type VideoInfo struct {
	Title string     // 标题
	Bvid  string     // BV号
	Aid   int64      // AV号
	Cid   int64      // 默认 CID（第一P）
	Pic   string     // 封面 URL
	Owner string     // UP主名称
	Pages []PageInfo // 分P列表
}

// DashStream 单个音视频流信息
type DashStream struct {
	ID        int    // 清晰度/音质 ID
	BaseURL   string // 下载地址
	Bandwidth int64  // 码率
	Codecs    string // 编码格式
}

// --- 1. 获取视频信息 ---

// viewResponse 对应 /x/web-interface/view 接口返回
type viewResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Aid   int64  `json:"aid"`
		Bvid  string `json:"bvid"`
		Title string `json:"title"`
		Pic   string `json:"pic"`
		Cid   int64  `json:"cid"`
		Owner struct {
			Name string `json:"name"`
		} `json:"owner"`
		Pages []struct {
			Cid  int64  `json:"cid"`
			Part string `json:"part"`
			Page int    `json:"page"`
		} `json:"pages"`
	} `json:"data"`
}

// GetVideoInfo 通过 BVID 获取视频基本信息
func GetVideoInfo(bvid string) (*VideoInfo, error) {
	apiURL := "https://api.bilibili.com/x/web-interface/view?bvid=" + bvid

	req, _ := http.NewRequest("GET", apiURL, nil)
	addCommonHeaders(req)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求视频信息失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result viewResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析视频信息失败: %w", err)
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("B站 API 错误: %d %s", result.Code, result.Message)
	}

	info := &VideoInfo{
		Title: result.Data.Title,
		Bvid:  result.Data.Bvid,
		Aid:   result.Data.Aid,
		Cid:   result.Data.Cid,
		Pic:   result.Data.Pic,
		Owner: result.Data.Owner.Name,
	}

	for _, p := range result.Data.Pages {
		info.Pages = append(info.Pages, PageInfo{
			Cid:  p.Cid,
			Part: p.Part,
			Page: p.Page,
		})
	}

	return info, nil
}

// --- 2. 获取播放地址 ---

// playURLResponse 对应 playurl 接口返回
type playURLResponse struct {
	Code int `json:"code"`
	Data struct {
		Dash struct {
			Video []struct {
				ID        int    `json:"id"`
				BaseURL   string `json:"base_url"`
				Bandwidth int64  `json:"bandwidth"`
				Codecs    string `json:"codecs"`
			} `json:"video"`
			Audio []struct {
				ID        int    `json:"id"`
				BaseURL   string `json:"base_url"`
				Bandwidth int64  `json:"bandwidth"`
				Codecs    string `json:"codecs"`
			} `json:"audio"`
		} `json:"dash"`
	} `json:"data"`
}

// GetPlayURL 获取 DASH 格式的最佳音视频流地址
func GetPlayURL(aid, cid int64) (video *DashStream, audio *DashStream, err error) {
	imgKey, subKey, err := crawler.GetWbiKeys()
	if err != nil {
		return nil, nil, fmt.Errorf("获取 Wbi 签名密钥失败: %w", err)
	}

	params := map[string]string{
		"avid":  strconv.FormatInt(aid, 10),
		"cid":   strconv.FormatInt(cid, 10),
		"fnval": "4048", // 请求 DASH 格式
		"fnver": "0",
		"fourk": "1",
		"qn":    "127", // 请求最高画质
	}

	signedParams := crawler.SignAndEncode(params, imgKey, subKey)
	apiURL := "https://api.bilibili.com/x/player/wbi/playurl?" + signedParams

	req, _ := http.NewRequest("GET", apiURL, nil)
	addCommonHeaders(req)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("请求播放地址失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result playURLResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, nil, fmt.Errorf("解析播放地址失败: %w", err)
	}

	if result.Code != 0 {
		return nil, nil, fmt.Errorf("playurl API 错误 (code=%d)", result.Code)
	}

	// 选取最高码率的视频流
	var bestVideo *DashStream
	for _, v := range result.Data.Dash.Video {
		if bestVideo == nil || v.Bandwidth > bestVideo.Bandwidth {
			bestVideo = &DashStream{
				ID:        v.ID,
				BaseURL:   v.BaseURL,
				Bandwidth: v.Bandwidth,
				Codecs:    v.Codecs,
			}
		}
	}

	// 选取最高码率的音频流
	var bestAudio *DashStream
	for _, a := range result.Data.Dash.Audio {
		if bestAudio == nil || a.Bandwidth > bestAudio.Bandwidth {
			bestAudio = &DashStream{
				ID:        a.ID,
				BaseURL:   a.BaseURL,
				Bandwidth: a.Bandwidth,
				Codecs:    a.Codecs,
			}
		}
	}

	if bestVideo == nil {
		return nil, nil, fmt.Errorf("未找到可用的视频流")
	}
	if bestAudio == nil {
		return nil, nil, fmt.Errorf("未找到可用的音频流")
	}

	return bestVideo, bestAudio, nil
}

// --- 3. HTTP 文件下载 ---

// DownloadFile 下载文件到指定路径，带进度回调
func DownloadFile(url, savePath string, onProgress func(downloaded, total int64)) error {
	req, _ := http.NewRequest("GET", url, nil)
	addCommonHeaders(req)

	client := &http.Client{Timeout: 30 * time.Minute} // 大文件需要长超时
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("请求下载地址失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("HTTP 状态码异常: %d", resp.StatusCode)
	}

	totalSize := resp.ContentLength

	// 创建目录
	dir := filepath.Dir(savePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	file, err := os.Create(savePath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()

	buf := make([]byte, 256*1024) // 256KB buffer
	var downloaded int64

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := file.Write(buf[:n])
			if writeErr != nil {
				return fmt.Errorf("写入文件失败: %w", writeErr)
			}
			downloaded += int64(n)
			if onProgress != nil {
				onProgress(downloaded, totalSize)
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
func downloadSinglePage(info *VideoInfo, page PageInfo, outputDir, ffmpegPath, filePrefix string, onStatus func(msg string)) error {
	// 获取播放地址
	onStatus(fmt.Sprintf("正在获取 P%d 播放地址...", page.Page))
	videoStream, audioStream, err := GetPlayURL(info.Aid, page.Cid)
	if err != nil {
		return fmt.Errorf("获取 P%d 播放地址失败: %w", page.Page, err)
	}

	// 准备文件路径
	videoTmp := filepath.Join(outputDir, filePrefix+"_video.m4s")
	audioTmp := filepath.Join(outputDir, filePrefix+"_audio.m4s")
	finalOutput := filepath.Join(outputDir, filePrefix+".mp4")

	// 下载视频流
	onStatus(fmt.Sprintf("正在下载 P%d 视频流...", page.Page))
	if err := DownloadFile(videoStream.BaseURL, videoTmp, func(downloaded, total int64) {
		if total > 0 {
			pct := float64(downloaded) / float64(total) * 100
			onStatus(fmt.Sprintf("P%d 视频: %.1f%% (%d/%d MB)", page.Page, pct, downloaded/1024/1024, total/1024/1024))
		}
	}); err != nil {
		return fmt.Errorf("下载 P%d 视频流失败: %w", page.Page, err)
	}

	// 下载音频流
	onStatus(fmt.Sprintf("正在下载 P%d 音频流...", page.Page))
	if err := DownloadFile(audioStream.BaseURL, audioTmp, func(downloaded, total int64) {
		if total > 0 {
			pct := float64(downloaded) / float64(total) * 100
			onStatus(fmt.Sprintf("P%d 音频: %.1f%% (%d/%d MB)", page.Page, pct, downloaded/1024/1024, total/1024/1024))
		}
	}); err != nil {
		os.Remove(videoTmp)
		return fmt.Errorf("下载 P%d 音频流失败: %w", page.Page, err)
	}

	// ffmpeg 合并
	onStatus(fmt.Sprintf("正在合并 P%d 音视频...", page.Page))
	ffmpegCmd := "ffmpeg"
	if ffmpegPath != "" {
		ffmpegCmd = ffmpegPath
	}

	cmd := exec.Command(ffmpegCmd,
		"-i", videoTmp,
		"-i", audioTmp,
		"-c", "copy",
		"-y",
		"-loglevel", "error",
		finalOutput,
	)

	output, err := cmd.CombinedOutput()
	os.Remove(videoTmp)
	os.Remove(audioTmp)

	if err != nil {
		return fmt.Errorf("P%d ffmpeg 合并失败: %v | %s", page.Page, err, string(output))
	}

	return nil
}

// sanitizeFilename 清理文件名中的非法字符
func sanitizeFilename(name string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '\\', '/', '*', '?', '"', '<', '>', '|', ':':
			return '_'
		}
		return r
	}, name)
}

// DownloadPages 下载指定的分P列表（用于用户选择部分分P下载）
func DownloadPages(bvid string, pages []PageInfo, outputDir, ffmpegPath string, onStatus func(msg string)) error {
	if onStatus == nil {
		onStatus = func(msg string) {}
	}

	onStatus("正在获取视频信息...")
	info, err := GetVideoInfo(bvid)
	if err != nil {
		return fmt.Errorf("获取视频信息失败: %w", err)
	}
	onStatus(fmt.Sprintf("视频: %s (下载 %d/%d P)", info.Title, len(pages), len(info.Pages)))

	safeTitle := regexp.MustCompile(`[\\/*?:"<>|]`).ReplaceAllString(info.Title, "_")
	multiPage := len(info.Pages) > 1

	for _, page := range pages {
		var filePrefix string
		if multiPage {
			safePart := sanitizeFilename(page.Part)
			filePrefix = fmt.Sprintf("%s_P%02d_%s", safeTitle, page.Page, safePart)
		} else {
			filePrefix = safeTitle
		}

		if err := downloadSinglePage(info, page, outputDir, ffmpegPath, filePrefix, onStatus); err != nil {
			return err
		}
		onStatus(fmt.Sprintf("P%d 下载完成: %s.mp4", page.Page, filePrefix))
	}

	if info.Pic != "" {
		coverPath := filepath.Join(outputDir, safeTitle+".jpg")
		_ = DownloadFile(info.Pic, coverPath, nil)
	}

	onStatus(fmt.Sprintf("下载完成，共 %d P", len(pages)))
	return nil
}

// DownloadVideo 完整的视频下载流程：获取信息 -> 下载所有分P -> 合并
func DownloadVideo(bvid, outputDir, ffmpegPath string, onStatus func(msg string)) error {
	if onStatus == nil {
		onStatus = func(msg string) {} // 空回调
	}

	// 1. 获取视频信息
	onStatus("正在获取视频信息...")
	info, err := GetVideoInfo(bvid)
	if err != nil {
		return fmt.Errorf("获取视频信息失败: %w", err)
	}
	onStatus(fmt.Sprintf("视频: %s (UP主: %s, 共 %d P)", info.Title, info.Owner, len(info.Pages)))

	safeTitle := regexp.MustCompile(`[\\/*?:"<>|]`).ReplaceAllString(info.Title, "_")

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

		if err := downloadSinglePage(info, page, outputDir, ffmpegPath, filePrefix, onStatus); err != nil {
			return err
		}

		onStatus(fmt.Sprintf("P%d 下载完成: %s.mp4", page.Page, filePrefix))
	}

	// 3. 下载封面（仅一次）
	if info.Pic != "" {
		coverPath := filepath.Join(outputDir, safeTitle+".jpg")
		_ = DownloadFile(info.Pic, coverPath, nil)
	}

	if multiPage {
		onStatus(fmt.Sprintf("全部 %d P 下载完成!", len(info.Pages)))
	} else {
		onStatus(fmt.Sprintf("下载完成: %s", filepath.Join(outputDir, safeTitle+".mp4")))
	}
	return nil
}
