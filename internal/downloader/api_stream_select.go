package downloader

import (
	"fmt"
	"sort"
	"strings"
)

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
