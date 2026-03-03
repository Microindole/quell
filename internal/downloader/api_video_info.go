package downloader

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"quell/internal/crawler"
)

// GetVideoInfo 通过 BVID 获取视频基本信息。
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
		return nil, fmt.Errorf("b站 API 错误: %d %s", result.Code, result.Message)
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
			Cid:      p.Cid,
			Part:     p.Part,
			Page:     p.Page,
			Duration: p.Duration,
		})
	}
	return info, nil
}

func fetchPlayURL(aid, cid int64, qn int) (*playURLResponse, error) {
	result, err := fetchPlayURLWithFnval(aid, cid, qn, "4048")
	if err != nil {
		return nil, err
	}
	if len(result.Data.Dash.Video) > 0 {
		return result, nil
	}
	fallback, err := fetchPlayURLWithFnval(aid, cid, qn, "0")
	if err != nil {
		return nil, err
	}
	return fallback, nil
}

func fetchPlayURLWithFnval(aid, cid int64, qn int, fnval string) (*playURLResponse, error) {
	imgKey, subKey, err := crawler.GetWbiKeys()
	if err != nil {
		return nil, fmt.Errorf("获取 Wbi 签名密钥失败: %w", err)
	}

	qnVal := "127"
	if qn > 0 {
		qnVal = strconv.Itoa(qn)
	}
	params := map[string]string{
		"avid":  strconv.FormatInt(aid, 10),
		"cid":   strconv.FormatInt(cid, 10),
		"fnval": fnval,
		"fnver": "0",
		"fourk": "1",
		"qn":    qnVal,
	}

	apiURL := "https://api.bilibili.com/x/player/wbi/playurl?" + crawler.SignAndEncode(params, imgKey, subKey)
	req, _ := http.NewRequest("GET", apiURL, nil)
	addCommonHeaders(req)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求播放地址失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result playURLResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析播放地址失败: %w", err)
	}
	if result.Code != 0 {
		return nil, fmt.Errorf("playurl API 错误 (code=%d, msg=%s)", result.Code, result.Message)
	}
	return &result, nil
}

func mapVideoStreams(result *playURLResponse) []DashStream {
	streams := make([]DashStream, 0, len(result.Data.Dash.Video))
	for _, v := range result.Data.Dash.Video {
		streams = append(streams, DashStream{
			ID:        v.ID,
			BaseURL:   v.BaseURL,
			Bandwidth: v.Bandwidth,
			Codecs:    v.Codecs,
		})
	}
	return streams
}

func mapAudioStreams(result *playURLResponse) []DashStream {
	streams := make([]DashStream, 0, len(result.Data.Dash.Audio))
	for _, a := range result.Data.Dash.Audio {
		streams = append(streams, DashStream{
			ID:        a.ID,
			BaseURL:   a.BaseURL,
			Bandwidth: a.Bandwidth,
			Codecs:    a.Codecs,
		})
	}
	return streams
}

func mapDurlVideo(result *playURLResponse) *DashStream {
	if len(result.Data.Durl) == 0 {
		return nil
	}
	first := result.Data.Durl[0]
	if first.URL == "" {
		return nil
	}
	qid := 0
	if len(result.Data.AcceptQuality) > 0 {
		qid = result.Data.AcceptQuality[0]
	}
	return &DashStream{
		ID:        qid,
		BaseURL:   first.URL,
		Bandwidth: first.Size,
		Codecs:    "mixed",
	}
}

func inferTrialSeconds(result *playURLResponse) int64 {
	if len(result.Data.Durl) == 0 {
		return 0
	}
	length := result.Data.Durl[0].Length
	if length <= 0 {
		return 0
	}
	return length / 1000
}

func inferPlayableMs(result *playURLResponse) int64 {
	if result.Data.Timelength > 0 {
		return result.Data.Timelength
	}
	if len(result.Data.Durl) > 0 && result.Data.Durl[0].Length > 0 {
		return result.Data.Durl[0].Length
	}
	return 0
}

func buildDebugSummary(result *playURLResponse, mode string) string {
	return fmt.Sprintf(
		"mode=%s, is_preview=%d, timelength_ms=%d, dash_v=%d, dash_a=%d, durl=%d",
		mode,
		result.Data.IsPreview,
		result.Data.Timelength,
		len(result.Data.Dash.Video),
		len(result.Data.Dash.Audio),
		len(result.Data.Durl),
	)
}
