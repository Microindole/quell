package downloader

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
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
			Cid      int64  `json:"cid"`
			Part     string `json:"part"`
			Page     int    `json:"page"`
			Duration int64  `json:"duration"`
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

// playURLResponse 对应 playurl 接口返回
type playURLResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		AcceptQuality []int    `json:"accept_quality"`
		AcceptDesc    []string `json:"accept_description"`
		IsPreview     int      `json:"is_preview"`
		Timelength    int64    `json:"timelength"`
		Durl          []struct {
			URL       string   `json:"url"`
			Size      int64    `json:"size"`
			Order     int      `json:"order"`
			Length    int64    `json:"length"`
			BackupURL []string `json:"backup_url"`
		} `json:"durl"`
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

func fetchPlayURL(aid, cid int64, qn int) (*playURLResponse, error) {
	result, err := fetchPlayURLWithFnval(aid, cid, qn, "4048")
	if err != nil {
		return nil, err
	}
	if len(result.Data.Dash.Video) > 0 {
		return result, nil
	}
	// 回退：部分视频（如充电专属）不返回 DASH，只返回 durl 直链。
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

	signedParams := crawler.SignAndEncode(params, imgKey, subKey)
	apiURL := "https://api.bilibili.com/x/player/wbi/playurl?" + signedParams

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
	// durl length 通常为毫秒
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

func normalizeCodec(codec string) string {
	c := strings.ToLower(codec)
	switch {
	case strings.HasPrefix(c, "avc"):
		return "avc"
	case strings.HasPrefix(c, "hev"), strings.HasPrefix(c, "hvc"):
		return "hevc"
	case strings.HasPrefix(c, "av01"):
		return "av1"
	default:
		return c
	}
}

func codecLabel(codec string) string {
	switch normalizeCodec(codec) {
	case "avc":
		return "AVC (H.264)"
	case "hevc":
		return "HEVC (H.265)"
	case "av1":
		return "AV1"
	default:
		return strings.ToUpper(codec)
	}
}

func qualityLabel(qid int) string {
	switch qid {
	case 127:
		return "8K 超高清"
	case 126:
		return "杜比视界"
	case 125:
		return "HDR 真彩"
	case 120:
		return "4K 超清"
	case 116:
		return "1080P60 高帧率"
	case 112:
		return "1080P+ 高码率"
	case 80:
		return "1080P 高清"
	case 74:
		return "720P60 高帧率"
	case 64:
		return "720P 高清"
	case 32:
		return "480P 清晰"
	case 16:
		return "360P 流畅"
	default:
		return fmt.Sprintf("清晰度 %d", qid)
	}
}

func qualityRank(qid int) int {
	switch qid {
	case 127:
		return 1100
	case 126:
		return 1090
	case 125:
		return 1080
	case 120:
		return 1070
	case 116:
		return 1060
	case 112:
		return 1050
	case 80:
		return 1040
	case 74:
		return 1030
	case 64:
		return 1020
	case 32:
		return 1010
	case 16:
		return 1000
	default:
		return qid
	}
}

func codecRank(codec string) int {
	switch normalizeCodec(codec) {
	case "av1":
		return 30
	case "hevc":
		return 20
	case "avc":
		return 10
	default:
		return 0
	}
}

// ListVideoStreamMetas 返回当前分P可选的视频清晰度/编码组合。
func ListVideoStreamMetas(aid, cid int64) ([]StreamMeta, error) {
	result, err := fetchPlayURL(aid, cid, 127)
	if err != nil {
		return nil, err
	}

	videoStreams := mapVideoStreams(result)
	if len(videoStreams) == 0 {
		if durl := mapDurlVideo(result); durl != nil {
			label := "兼容直链"
			if durl.ID > 0 {
				label = qualityLabel(durl.ID) + "（兼容直链）"
			}
			return []StreamMeta{
				{
					QualityID:    durl.ID,
					QualityLabel: label,
					Codec:        "mixed",
					CodecLabel:   "音视频直链",
					Bandwidth:    durl.Bandwidth,
				},
			}, nil
		}
		return nil, fmt.Errorf("未找到可用的视频流（可能是权限受限/充电专属，或当前账号无播放权限）")
	}

	metaByKey := map[string]StreamMeta{}
	for _, v := range videoStreams {
		key := fmt.Sprintf("%d|%s", v.ID, normalizeCodec(v.Codecs))
		meta := StreamMeta{
			QualityID:    v.ID,
			QualityLabel: qualityLabel(v.ID),
			Codec:        normalizeCodec(v.Codecs),
			CodecLabel:   codecLabel(v.Codecs),
			Bandwidth:    v.Bandwidth,
		}
		if old, ok := metaByKey[key]; !ok || meta.Bandwidth > old.Bandwidth {
			metaByKey[key] = meta
		}
	}

	metas := make([]StreamMeta, 0, len(metaByKey))
	for _, meta := range metaByKey {
		metas = append(metas, meta)
	}

	sort.SliceStable(metas, func(i, j int) bool {
		ri, rj := qualityRank(metas[i].QualityID), qualityRank(metas[j].QualityID)
		if ri != rj {
			return ri > rj
		}
		ci, cj := codecRank(metas[i].Codec), codecRank(metas[j].Codec)
		if ci != cj {
			return ci > cj
		}
		return metas[i].Bandwidth > metas[j].Bandwidth
	})
	return metas, nil
}

func pickBestAudio(audioStreams []DashStream) *DashStream {
	var bestAudio *DashStream
	for _, a := range audioStreams {
		aa := a
		if bestAudio == nil || aa.Bandwidth > bestAudio.Bandwidth {
			bestAudio = &aa
		}
	}
	return bestAudio
}

func selectVideoStream(videoStreams []DashStream, pref DownloadPreference) *DashStream {
	if len(videoStreams) == 0 {
		return nil
	}

	candidates := make([]DashStream, 0, len(videoStreams))
	for _, v := range videoStreams {
		if pref.QualityID > 0 && v.ID != pref.QualityID {
			continue
		}
		candidates = append(candidates, v)
	}
	if len(candidates) == 0 {
		candidates = videoStreams
	}

	if pref.Codec != "" {
		codecFiltered := make([]DashStream, 0, len(candidates))
		for _, v := range candidates {
			if normalizeCodec(v.Codecs) == normalizeCodec(pref.Codec) {
				codecFiltered = append(codecFiltered, v)
			}
		}
		if len(codecFiltered) > 0 {
			candidates = codecFiltered
		}
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		ri, rj := qualityRank(candidates[i].ID), qualityRank(candidates[j].ID)
		if ri != rj {
			return ri > rj
		}
		if normalizeCodec(pref.Codec) != "" {
			pi := 0
			pj := 0
			if normalizeCodec(candidates[i].Codecs) == normalizeCodec(pref.Codec) {
				pi = 1
			}
			if normalizeCodec(candidates[j].Codecs) == normalizeCodec(pref.Codec) {
				pj = 1
			}
			if pi != pj {
				return pi > pj
			}
		}
		ci, cj := codecRank(candidates[i].Codecs), codecRank(candidates[j].Codecs)
		if ci != cj {
			return ci > cj
		}
		return candidates[i].Bandwidth > candidates[j].Bandwidth
	})
	best := candidates[0]
	return &best
}

// GetPlayURL 获取 DASH 格式的音视频流地址，并按偏好选择视频流。
func GetPlayURL(aid, cid int64, expectedDurationSec int64, pref DownloadPreference) (*PlayURLSelection, error) {
	result, err := fetchPlayURL(aid, cid, pref.QualityID)
	if err != nil {
		return nil, err
	}

	videoStreams := mapVideoStreams(result)
	audioStreams := mapAudioStreams(result)
	mode := "dash"
	if len(videoStreams) == 0 && len(result.Data.Durl) > 0 {
		mode = "durl"
	}

	isPreview := result.Data.IsPreview == 1
	trialSeconds := inferTrialSeconds(result)
	if isPreview {
		return nil, fmt.Errorf("当前视频仅返回试看流，无法下载完整内容（%s, 试看时长约 %d 秒）", buildDebugSummary(result, mode), trialSeconds)
	}
	// 兜底校验：部分充电/权限场景 is_preview=0，但返回的可播时长明显短于分P标称时长。
	// 允许 5 秒误差，避免因片头片尾裁切导致误判。
	if expectedDurationSec > 0 {
		expectedMs := expectedDurationSec * 1000
		playableMs := inferPlayableMs(result)
		if playableMs > 0 && playableMs+5000 < expectedMs {
			return nil, fmt.Errorf(
				"当前视频疑似仅返回试看流，无法下载完整内容（%s, 返回时长约 %d 秒，标称时长约 %d 秒）",
				buildDebugSummary(result, mode),
				playableMs/1000,
				expectedDurationSec,
			)
		}
	}

	bestVideo := selectVideoStream(videoStreams, pref)
	bestAudio := pickBestAudio(audioStreams)

	if bestVideo == nil {
		if durl := mapDurlVideo(result); durl != nil {
			return &PlayURLSelection{
				Video:        durl,
				Audio:        nil,
				Mode:         "durl",
				IsPreview:    false,
				TrialSeconds: 0,
				Debug:        buildDebugSummary(result, "durl"),
			}, nil
		}
		return nil, fmt.Errorf("未找到可用的视频流（%s）", buildDebugSummary(result, mode))
	}
	if bestAudio == nil {
		// 允许无独立音轨：后续走直链下载，不做 ffmpeg 合并。
		return &PlayURLSelection{
			Video:        bestVideo,
			Audio:        nil,
			Mode:         mode,
			IsPreview:    false,
			TrialSeconds: 0,
			Debug:        buildDebugSummary(result, mode),
		}, nil
	}

	return &PlayURLSelection{
		Video:        bestVideo,
		Audio:        bestAudio,
		Mode:         "dash",
		IsPreview:    false,
		TrialSeconds: 0,
		Debug:        buildDebugSummary(result, "dash"),
	}, nil
}
