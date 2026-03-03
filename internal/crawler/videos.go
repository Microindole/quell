package crawler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func normalizeVideoOrder(order string) string {
	switch strings.ToLower(strings.TrimSpace(order)) {
	case "click", "hot":
		return "click"
	default:
		return "pubdate"
	}
}

func getUserVideosOnce(mid string, pn, ps int, order string, withSess bool) ([]BiliVideoMeta, int, int, string, error) {
	imgKey, subKey, err := GetWbiKeys()
	if err != nil {
		return nil, 0, 0, "", fmt.Errorf("failed to get wbi keys: %w", err)
	}
	if ps <= 0 {
		ps = 30
	}

	params := map[string]string{
		"mid":   mid,
		"ps":    strconv.Itoa(ps),
		"tid":   "0",
		"order": normalizeVideoOrder(order),
		"pn":    strconv.Itoa(pn),
	}
	apiURL := "https://api.bilibili.com/x/space/wbi/arc/search?" + SignAndEncode(params, imgKey, subKey)

	req, _ := http.NewRequest("GET", apiURL, nil)
	addCommonHeadersWithSess(req, withSess)
	req.Header.Set("Referer", "https://space.bilibili.com/"+mid)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, 0, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, 0, 0, "", fmt.Errorf("status code %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result arcSearchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, 0, 0, "", fmt.Errorf("解析响应失败: %w (原始响应: %.200s)", err, string(body))
	}
	if result.Code != 0 {
		return nil, 0, result.Code, result.Message, nil
	}
	if len(result.Data.List.Vlist) == 0 {
		return nil, 0, result.Code, result.Message, fmt.Errorf("该用户 (mid=%s) 没有公开视频，或被 B 站风控拦截 (原始响应: %.300s)", mid, string(body))
	}

	videos := make([]BiliVideoMeta, 0, len(result.Data.List.Vlist))
	for _, v := range result.Data.List.Vlist {
		pic := v.Pic
		if strings.HasPrefix(pic, "//") {
			pic = "https:" + pic
		}
		videos = append(videos, BiliVideoMeta{
			Title:    stripHTMLTags(v.Title),
			Bvid:     v.Bvid,
			Aid:      v.Aid,
			Pic:      pic,
			Subtitle: v.Subtitle,
			Length:   v.Length,
			Created:  v.Created,
			Play:     parseStatAny(v.Play),
			Danmaku:  parseStatAny(v.VideoReview),
		})
	}
	return videos, result.Data.Page.Xcount, result.Code, result.Message, nil
}

func GetUserVideos(mid string, pn, ps int, order string) ([]BiliVideoMeta, int, error) {
	videos, total, code, msg, err := getUserVideosOnce(mid, pn, ps, order, true)
	if err == nil && code == 0 {
		return videos, total, nil
	}
	if err != nil {
		return nil, 0, err
	}

	if code == -352 {
		resetWbiCache()
		videos2, total2, code2, msg2, err2 := getUserVideosOnce(mid, pn, ps, order, false)
		if err2 == nil && code2 == 0 {
			return videos2, total2, nil
		}
		if err2 != nil {
			return nil, 0, err2
		}
		return nil, 0, fmt.Errorf("b站 API 错误 (code=%d): %s（已重试一次）", code2, msg2)
	}

	return nil, 0, fmt.Errorf("b站 API 错误 (code=%d): %s", code, msg)
}

func stripHTMLTags(s string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	return strings.TrimSpace(re.ReplaceAllString(s, ""))
}

func formatLength(sec int64) string {
	if sec <= 0 {
		return "00:00"
	}
	h := sec / 3600
	m := (sec % 3600) / 60
	s := sec % 60
	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}

func parseDurationAny(v interface{}) int64 {
	switch d := v.(type) {
	case float64:
		return int64(d)
	case int64:
		return d
	case int:
		return int64(d)
	case string:
		s := strings.TrimSpace(d)
		if s == "" {
			return 0
		}
		if strings.Contains(s, ":") {
			parts := strings.Split(s, ":")
			total := int64(0)
			for _, p := range parts {
				n, err := strconv.Atoi(strings.TrimSpace(p))
				if err != nil {
					return 0
				}
				total = total*60 + int64(n)
			}
			return total
		}
		n, err := strconv.Atoi(s)
		if err != nil {
			return 0
		}
		return int64(n)
	default:
		return 0
	}
}

func formatCountInt64(n int64) string {
	if n < 0 {
		return "--"
	}
	if n >= 100000000 {
		return fmt.Sprintf("%.1f亿", float64(n)/100000000)
	}
	if n >= 10000 {
		return fmt.Sprintf("%.1f万", float64(n)/10000)
	}
	return strconv.FormatInt(n, 10)
}

func parseStatAny(v interface{}) string {
	switch x := v.(type) {
	case float64:
		return formatCountInt64(int64(x))
	case int64:
		return formatCountInt64(x)
	case int:
		return formatCountInt64(int64(x))
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return "--"
		}
		if n, err := strconv.ParseFloat(s, 64); err == nil {
			return formatCountInt64(int64(n))
		}
		return s
	default:
		return "--"
	}
}

func parseInt64Any(v interface{}) int64 {
	switch x := v.(type) {
	case float64:
		return int64(x)
	case int64:
		return x
	case int:
		return int64(x)
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return 0
		}
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return int64(f)
		}
		return 0
	default:
		return 0
	}
}

func fetchVideoByID(keyword string) ([]BiliVideoMeta, int, error) {
	id := strings.TrimSpace(keyword)
	lower := strings.ToLower(id)
	switch {
	case strings.HasPrefix(lower, "bv"):
		id = "BV" + id[2:]
	case strings.HasPrefix(lower, "av"):
		id = "av" + id[2:]
	case strings.HasPrefix(lower, "cv"):
		return nil, 0, fmt.Errorf("CV 专栏暂不支持下载，仅支持视频（BV/AV）")
	default:
		return nil, 0, fmt.Errorf("不支持的ID格式: %s", keyword)
	}

	params := url.Values{}
	if strings.HasPrefix(strings.ToLower(id), "bv") {
		params.Set("bvid", id)
	} else {
		params.Set("aid", strings.TrimPrefix(strings.ToLower(id), "av"))
	}

	req, _ := http.NewRequest("GET", "https://api.bilibili.com/x/web-interface/view?"+params.Encode(), nil)
	addCommonHeaders(req)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Title    string `json:"title"`
			Bvid     string `json:"bvid"`
			Aid      int    `json:"aid"`
			Pic      string `json:"pic"`
			Duration int64  `json:"duration"`
			Pubdate  int64  `json:"pubdate"`
			Stat     struct {
				View    int64 `json:"view"`
				Danmaku int64 `json:"danmaku"`
			} `json:"stat"`
			Owner struct {
				Name string `json:"name"`
			} `json:"owner"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, 0, fmt.Errorf("解析视频详情失败: %w", err)
	}
	if result.Code != 0 {
		return nil, 0, fmt.Errorf("b站 API 错误 (code=%d): %s", result.Code, result.Message)
	}

	meta := BiliVideoMeta{
		Title:    result.Data.Title,
		Bvid:     result.Data.Bvid,
		Aid:      result.Data.Aid,
		Pic:      result.Data.Pic,
		Subtitle: result.Data.Owner.Name,
		Length:   formatLength(result.Data.Duration),
		Created:  result.Data.Pubdate,
		Play:     formatCountInt64(result.Data.Stat.View),
		Danmaku:  formatCountInt64(result.Data.Stat.Danmaku),
	}
	return []BiliVideoMeta{meta}, 1, nil
}

func SearchVideos(keyword string, pn, ps int, order string) ([]BiliVideoMeta, int, error) {
	kw := strings.TrimSpace(keyword)
	if kw == "" {
		return nil, 0, fmt.Errorf("关键词不能为空")
	}

	lower := strings.ToLower(kw)
	if strings.HasPrefix(lower, "bv") || strings.HasPrefix(lower, "av") || strings.HasPrefix(lower, "cv") {
		return fetchVideoByID(kw)
	}
	if pn <= 0 {
		pn = 1
	}
	if ps <= 0 {
		ps = 30
	}

	imgKey, subKey, err := GetWbiKeys()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get wbi keys: %w", err)
	}

	params := map[string]string{
		"keyword":       kw,
		"search_type":   "video",
		"order":         normalizeVideoOrder(order),
		"page":          strconv.Itoa(pn),
		"page_size":     strconv.Itoa(ps),
		"platform":      "pc",
		"highlight":     "0",
		"single_column": "0",
	}
	req, _ := http.NewRequest("GET", "https://api.bilibili.com/x/web-interface/wbi/search/type?"+SignAndEncode(params, imgKey, subKey), nil)
	addCommonHeaders(req)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			NumResults int `json:"numResults"`
			Result     []struct {
				Title    string      `json:"title"`
				Bvid     string      `json:"bvid"`
				Aid      int         `json:"aid"`
				Pic      string      `json:"pic"`
				Duration interface{} `json:"duration"`
				Pubdate  int64       `json:"pubdate"`
				Author   string      `json:"author"`
				Play     interface{} `json:"play"`
				Danmaku  interface{} `json:"video_review"`
			} `json:"result"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, 0, fmt.Errorf("解析视频搜索结果失败: %w", err)
	}
	if result.Code != 0 {
		return nil, 0, fmt.Errorf("b站 API 错误 (code=%d): %s", result.Code, result.Message)
	}

	videos := make([]BiliVideoMeta, 0, len(result.Data.Result))
	for _, v := range result.Data.Result {
		pic := v.Pic
		if strings.HasPrefix(pic, "//") {
			pic = "https:" + pic
		}
		videos = append(videos, BiliVideoMeta{
			Title:    stripHTMLTags(v.Title),
			Bvid:     v.Bvid,
			Aid:      v.Aid,
			Pic:      pic,
			Subtitle: v.Author,
			Length:   formatLength(parseDurationAny(v.Duration)),
			Created:  v.Pubdate,
			Play:     parseStatAny(v.Play),
			Danmaku:  parseStatAny(v.Danmaku),
		})
	}
	return videos, result.Data.NumResults, nil
}
