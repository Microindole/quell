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
	"time"

	"quell/internal/domain"
)

// MergeConfig 控制合并行为
type MergeConfig struct {
	FFmpegPath   string
	OutputFormat string           // "mp4" 或 "mkv"，默认 "mp4"
	OutputDir    string           // 合并结果输出目录，若为空则输出到任务目录
	OnProgress   func(msg string) // 进度回调，传递当前处理的时间戳或状态
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
		return "", fmt.Errorf("未找到可合并性 m4s 文件对")
	}

	coverPath := findCoverImage(task.Dir)

	// 确定输出目录
	outDir := cfg.OutputDir
	if outDir == "" {
		outDir = task.Dir
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return "", fmt.Errorf("创建输出目录失败: %w", err)
	}

	var lastOutput string
	for _, pair := range pairs {
		outName := GetOutputName(task, pair.Page)
		outPath := filepath.Join(outDir, outName+"."+cfg.OutputFormat)
		
		if cfg.OnProgress != nil {
			cfg.OnProgress(fmt.Sprintf("正在处理: %s", outName))
		}

		if err := mergePair(pair, outPath, coverPath, ffmpeg, cfg.OutputFormat, cfg.OnProgress, task.Info); err != nil {
			return "", fmt.Errorf("合并第 %d 分P失败: %w", pair.Page, err)
		}
		lastOutput = outPath
	}
	return lastOutput, nil
}

// GetOutputName 根据任务信息生成规范的输出文件名（不含后缀）
func GetOutputName(task domain.VideoTask, page int) string {
	mainTitle := task.Info.GroupTitle
	if mainTitle == "" {
		mainTitle = task.Info.Title
	}
	if mainTitle == "" {
		mainTitle = task.FolderName
	}
	safeMainTitle := sanitizeFilename(mainTitle)

	// 如果没有明确的分P，或者是单集，直接返回主标题
	// 注意：findM4SPairs 找到的对数 len(pairs) 这里拿不到，
	// 但我们可以根据 task.Info.P 或传入的 page 来判断
	if page <= 1 && task.Info.P <= 1 {
		return safeMainTitle
	}

	pNum := page
	if pNum == 0 { pNum = task.Info.P }
	if pNum == 0 { pNum = 1 }
	
	subTitle := ""
	if task.Info.GroupTitle != "" && task.Info.Title != "" && task.Info.Title != task.Info.GroupTitle {
		subTitle = "." + sanitizeFilename(task.Info.Title)
	}
	return fmt.Sprintf("%s.P%d%s", safeMainTitle, pNum, subTitle)
}

// m4sPair 表示一对视频+音频 m4s 文件
type m4sPair struct {
	Page  int
	Video string
	Audio string
}

func mergePair(pair m4sPair, outPath, coverPath, ffmpegPath, format string, onProgress func(string), info domain.BiliVideoInfo) error {
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

	args := buildFFmpegArgs(tmpVideo, tmpAudio, coverPath, outPath, format, info)
	cmd := exec.Command(ffmpegPath, args...)

	if onProgress == nil {
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("FFmpeg 错误: %v\n输出:\n%s", err, string(output))
		}
		return nil
	}

	// 进阶：解析 stderr 获取进度
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	// 解析 time=00:00:00.00 格式
	re := regexp.MustCompile(`time=(\d+:\d+:\d+\.\d+)`)
	buf := make([]byte, 1024)
	for {
		n, err := stderr.Read(buf)
		if n > 0 {
			line := string(buf[:n])
			if m := re.FindStringSubmatch(line); m != nil {
				onProgress(m[1])
			}
		}
		if err != nil {
			break
		}
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("FFmpeg 运行失败: %w", err)
	}
	return nil
}

func buildFFmpegArgs(videoPath, audioPath, coverPath, outPath, format string, info domain.BiliVideoInfo) []string {
	// 1. 全局选项
	args := []string{"-y"}

	// 2. 输入文件 (所有的 -i 必须排在前面)
	args = append(args, "-i", videoPath, "-i", audioPath)
	
	hasCover := coverPath != ""
	if hasCover && format != "mkv" {
		// MP4 模式下封面作为第三个输入流
		args = append(args, "-i", coverPath)
	}

	// 3. 流映射与编码选项
	if hasCover {
		if format == "mkv" {
			// MKV 模式使用 -attach
			args = append(args, 
				"-map", "0:v", "-map", "1:a", 
				"-c", "copy",
				"-attach", coverPath, 
				"-metadata:s:t:0", "mimetype=image/jpeg",
			)
		} else {
			// MP4 模式映射三个输入流
			args = append(args, 
				"-map", "0:v", "-map", "1:a", "-map", "2:v", 
				"-c", "copy",
				"-disposition:v:1", "attached_pic",
			)
		}
	} else {
		args = append(args, "-map", "0:v", "-map", "1:a", "-c", "copy")
	}

	// 4. 输出元数据 (必须在所有输入之后)
	if info.Title != "" {
		args = append(args, "-metadata", "title="+info.Title)
	}
	if info.GroupTitle != "" && info.GroupTitle != info.Title {
		args = append(args, "-metadata", "album="+info.GroupTitle)
	}
	if info.Uname != "" {
		args = append(args, "-metadata", "author="+info.Uname, "-metadata", "artist="+info.Uname)
	}
	if info.Bvid != "" {
		args = append(args, "-metadata", "comment=Bilibili: "+info.Bvid)
	}
	if info.Pubdate > 0 {
		t := time.Unix(info.Pubdate, 0)
		args = append(args, "-metadata", "date="+t.Format("2006-01-02"))
	}

	// 5. 输出文件
	return append(args, outPath)
}

func sanitizeFilename(name string) string {
	return regexp.MustCompile(`[\\/*?:"<>|]`).ReplaceAllString(name, "_")
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
	type candidate struct {
		path    string
		quality int
	}
	// pageMap[page][0]=最优视频, [1]=最优音频
	pageMap := make(map[int][2]candidate)

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".m4s" {
			return nil
		}

		filename := d.Name()
		page, quality, ok := parseM4SName(filename)
		
		if !ok {
			// 尝试兼容新版客户端: video.m4s / audio.m4s
			// 通常这种结构下，quality 是父目录名，page 可能需要从更上层获取，这里暂时默认为 1
			// 或者从父目录名尝试解析 quality
			parentDir := filepath.Base(filepath.Dir(path))
			if filename == "video.m4s" {
				q, _ := strconv.Atoi(parentDir)
				page, quality, ok = 1, q, true
			} else if filename == "audio.m4s" {
				q, _ := strconv.Atoi(parentDir)
				page, quality, ok = 1, q, true
				if quality == 0 { quality = 30280 } // 给音频一个默认的高 quality 标识
			}
		}

		if !ok {
			return nil
		}

		c := candidate{path: path, quality: quality}
		entry := pageMap[page]
		
		// 简单的逻辑：quality >= 30200 或者是文件名包含 audio
		isAudio := quality >= 30200 || filename == "audio.m4s"
		
		if isAudio { // 音频流
			if entry[1].path == "" || quality > entry[1].quality {
				entry[1] = c
			}
		} else { // 视频流
			if entry[0].path == "" || quality > entry[0].quality {
				entry[0] = c
			}
		}
		pageMap[page] = entry
		return nil
	})

	if err != nil {
		return nil, err
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

func BatchMerge(tasks []domain.VideoTask, cfg MergeConfig) {
	for i, task := range tasks {
		if task.Status == "完成" {
			continue
		}
		if cfg.OnProgress != nil {
			cfg.OnProgress(fmt.Sprintf("[%d/%d] 正在合并: %s", i+1, len(tasks), task.DisplayTitle()))
		}
		_, err := RunMerge(task, cfg)
		if err != nil && cfg.OnProgress != nil {
			cfg.OnProgress(fmt.Sprintf("合并失败: %s - %v", task.DisplayTitle(), err))
		}
	}
	if cfg.OnProgress != nil {
		cfg.OnProgress("批量合并完成")
	}
}
