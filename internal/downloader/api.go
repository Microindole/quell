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
