package engine

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"

	"quell/internal/domain"
)

// MergeConfig 控制合并行为
type MergeConfig struct {
	FFmpegPath   string
	OutputFormat string // "mp4" 或 "mkv"，默认 "mp4"
}

// RunMerge 用纯 Go + FFmpeg 将 B站缓存的 .m4s 合并为视频文件
func RunMerge(task domain.VideoTask, cfg MergeConfig) (string, error) {
	if cfg.OutputFormat == "" {
		cfg.OutputFormat = "mp4"
	}
	ffmpeg := cfg.FFmpegPath
	if ffmpeg == "" {
		ffmpeg = "ffmpeg"
	}

	pairs, err := findM4SPairs(task.Dir)
	if err != nil {
		return "", fmt.Errorf("查找 m4s 文件失败: %w", err)
	}
	if len(pairs) == 0 {
		return "", fmt.Errorf("未找到可合并的 m4s 文件对")
	}

	coverPath := findCoverImage(task.Dir)
	safeTitle := sanitizeFilename(task.DisplayTitle())

	var lastOutput string
	for _, pair := range pairs {
		outName := safeTitle
		if len(pairs) > 1 {
			outName = fmt.Sprintf("%s_p%d", safeTitle, pair.Page)
		}
		outPath := filepath.Join(task.Dir, outName+"."+cfg.OutputFormat)
		if err := mergePair(pair, outPath, coverPath, ffmpeg, cfg.OutputFormat); err != nil {
			return "", fmt.Errorf("合并第 %d 分P失败: %w", pair.Page, err)
		}
		lastOutput = outPath
	}
	return lastOutput, nil
}

// m4sPair 表示一对视频+音频 m4s 文件
type m4sPair struct {
	Page  int
	Video string
	Audio string
}

func mergePair(pair m4sPair, outPath, coverPath, ffmpegPath, format string) error {
	tmpVideo, err := stripBiliHeader(pair.Video)
	if err != nil {
		return fmt.Errorf("处理视频流失败: %w", err)
	}
	defer os.Remove(tmpVideo)

	tmpAudio, err := stripBiliHeader(pair.Audio)
	if err != nil {
		return fmt.Errorf("处理音频流失败: %w", err)
	}
	defer os.Remove(tmpAudio)

	args := buildFFmpegArgs(tmpVideo, tmpAudio, coverPath, outPath, format)
	cmd := exec.Command(ffmpegPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("FFmpeg 错误: %v\n输出:\n%s", err, string(output))
	}
	return nil
}

func buildFFmpegArgs(videoPath, audioPath, coverPath, outPath, format string) []string {
	args := []string{"-y", "-i", videoPath, "-i", audioPath}
	if coverPath != "" {
		if format == "mkv" {
			args = append(args,
				"-attach", coverPath,
				"-metadata:s:t:0", "mimetype=image/jpeg",
				"-map", "0:v", "-map", "1:a", "-c", "copy",
			)
		} else {
			// MP4: embed cover as attached picture stream
			args = append(args,
				"-i", coverPath,
				"-map", "0:v", "-map", "1:a", "-map", "2:v",
				"-c", "copy",
				"-disposition:v:1", "attached_pic",
			)
		}
	} else {
		args = append(args, "-map", "0:v", "-map", "1:a", "-c", "copy")
	}
	return append(args, outPath)
}

// stripBiliHeader 跳过 B站 .m4s 文件的私有头部，写到临时文件并返回路径
func stripBiliHeader(srcPath string) (string, error) {
	f, err := os.Open(srcPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// 读前 256 字节定位 ftyp 盒子
	probe := make([]byte, 256)
	n, _ := f.Read(probe)
	probe = probe[:n]

	// MP4 盒子结构：[4字节大小][ftyp 4字节]...
	// ftyp = 0x66 0x74 0x79 0x70，其前 4 字节是 size 字段
	ftypMarker := []byte{0x66, 0x74, 0x79, 0x70}
	idx := bytes.Index(probe, ftypMarker)
	if idx < 4 {
		return "", fmt.Errorf("未找到 ftyp 标记，可能不是有效的 m4s 文件: %s", filepath.Base(srcPath))
	}
	startOffset := int64(idx - 4)

	if _, err := f.Seek(startOffset, io.SeekStart); err != nil {
		return "", err
	}

	tmpPath := srcPath + ".stripped"
	tmp, err := os.Create(tmpPath)
	if err != nil {
		return "", err
	}
	defer tmp.Close()

	if _, err := io.Copy(tmp, f); err != nil {
		os.Remove(tmpPath)
		return "", err
	}
	return tmpPath, nil
}

// findM4SPairs 按分P分组，找出每分P的视频+音频对
func findM4SPairs(dir string) ([]m4sPair, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	type candidate struct {
		path    string
		quality int
	}
	// pageMap[page][0]=最优视频, [1]=最优音频
	pageMap := make(map[int][2]candidate)

	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".m4s" {
			continue
		}
		page, quality, ok := parseM4SName(e.Name())
		if !ok {
			continue
		}
		c := candidate{path: filepath.Join(dir, e.Name()), quality: quality}
		entry := pageMap[page]
		if quality >= 30200 { // 音频流
			if entry[1].path == "" || quality > entry[1].quality {
				entry[1] = c
			}
		} else { // 视频流
			if entry[0].path == "" || quality > entry[0].quality {
				entry[0] = c
			}
		}
		pageMap[page] = entry
	}

	var pairs []m4sPair
	for page, entry := range pageMap {
		if entry[0].path == "" || entry[1].path == "" {
			continue // 不完整的对，跳过
		}
		pairs = append(pairs, m4sPair{Page: page, Video: entry[0].path, Audio: entry[1].path})
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].Page < pairs[j].Page })
	return pairs, nil
}

// parseM4SName 解析 B站 .m4s 文件名，提取分P序号和画质代码
//
// 支持两种格式：
//   - [id]_p[N]-1-[quality].m4s  e.g. 25658989929_p1-1-30280.m4s
//   - [id]-[N]-[quality].m4s     e.g. 30964780940-1-30280.m4s
//
// quality >= 30200 为音频，< 30200 为视频
func parseM4SName(filename string) (page, quality int, ok bool) {
	// Pattern A: 含 _p 标记
	reA := regexp.MustCompile(`_p(\d+)-\d+-(\d+)\.m4s$`)
	if m := reA.FindStringSubmatch(filename); m != nil {
		page, _ = strconv.Atoi(m[1])
		quality, _ = strconv.Atoi(m[2])
		return page, quality, true
	}
	// Pattern B: 纯数字段
	reB := regexp.MustCompile(`-(\d+)-(\d+)\.m4s$`)
	if m := reB.FindStringSubmatch(filename); m != nil {
		page, _ = strconv.Atoi(m[1])
		quality, _ = strconv.Atoi(m[2])
		return page, quality, true
	}
	return 0, 0, false
}

// findCoverImage 优先返回 image.jpg，其次 group.jpg
func findCoverImage(dir string) string {
	for _, name := range []string{"image.jpg", "group.jpg"} {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func sanitizeFilename(name string) string {
	return regexp.MustCompile(`[\\/*?:"<>|]`).ReplaceAllString(name, "_")
}
