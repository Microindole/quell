package crawler

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Wbi 签名所需的混合密钥表
var mixinKeyEncTab = []int{
	46, 47, 18, 2, 53, 8, 23, 32, 15, 50, 10, 31, 58, 3, 45, 35, 27, 43, 5, 49,
	33, 9, 42, 19, 29, 28, 14, 39, 12, 38, 41, 13, 37, 48, 7, 16, 24, 55, 40,
	61, 26, 17, 0, 1, 60, 51, 30, 4, 22, 25, 54, 21, 56, 59, 6, 63, 57, 62, 11,
	36, 20, 34, 44, 52,
}

// 缓存一下 imgKey 和 subKey，避免每次都请求
var (
	cacheMutex    sync.Mutex
	cachedImgKey  string
	cachedSubKey  string
	cacheTime     time.Time
	cacheDuration = 10 * time.Minute
)

// BiliVideoMeta 是搜索返回的单个视频信息
type BiliVideoMeta struct {
	Title    string `json:"title"`
	Bvid     string `json:"bvid"`
	Aid      int    `json:"aid"`
	Pic      string `json:"pic"` // 封面
	Subtitle string `json:"subtitle"`
	Length   string `json:"length"` // 时长 "03:20"
	Created  int64  `json:"created"`
}

type arcSearchResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		List struct {
			Vlist []BiliVideoMeta `json:"vlist"`
		} `json:"list"`
		Page struct {
			Xcount int `json:"count"`
			Pn     int `json:"pn"`
			Ps     int `json:"ps"`
		} `json:"page"`
	} `json:"data"`
}

func GetUserVideos(mid string, pn int) ([]BiliVideoMeta, int, error) {
	imgKey, subKey, err := GetWbiKeys()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get wbi keys: %w", err)
	}

	params := map[string]string{
		"mid":              mid,
		"ps":               "30", // 每页 30 个
		"tid":              "0",
		"keyword":          "",
		"order":            "pubdate",
		"platform":         "web",
		"web_location":     "1550101",
		"dm_img_list":      "[]",
		"dm_img_str":       "V2ViR2wgMS4wIChPcGVuR0wgRVMgMi4wIENocm9taXVtKQ",
		"dm_cover_img_str": "QU5HTEUgKEludGVsLCBJbnRlbChSKSBVSEQgR3JhcGhpY3MgNzcwICgweDAwMDBBNzgwKSwgRC1DMy0xMS0tMTA7IDMyLjAuMTAxLjU5NzIp",
		"pn":               strconv.Itoa(pn),
	}

	signedParams := SignAndEncode(params, imgKey, subKey)
	apiURL := "https://api.bilibili.com/x/space/wbi/arc/search?" + signedParams

	req, _ := http.NewRequest("GET", apiURL, nil)
	// 必须要有 User-Agent，不然直接 403/412
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://space.bilibili.com/"+mid)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, 0, fmt.Errorf("status code %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result arcSearchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, 0, fmt.Errorf("解析响应失败: %w (原始响应: %.200s)", err, string(body))
	}

	if result.Code != 0 {
		return nil, 0, fmt.Errorf("B站 API 错误 (code=%d): %s", result.Code, result.Message)
	}

	if len(result.Data.List.Vlist) == 0 {
		return nil, 0, fmt.Errorf("该用户 (mid=%s) 没有公开视频，或被 B 站风控拦截 (原始响应: %.300s)", mid, string(body))
	}

	return result.Data.List.Vlist, result.Data.Page.Xcount, nil
}

// BiliUserMeta 搜索到的用户信息
type BiliUserMeta struct {
	Mid    int    `json:"mid"`
	Uname  string `json:"uname"`
	Usign  string `json:"usign"`  // 签名
	Fans   int    `json:"fans"`   // 粉丝数
	Verify bool   `json:"verify"` // 是否认证
	Upic   string `json:"upic"`   // 头像
}

type searchUserResponse struct {
	Code int `json:"code"`
	Data struct {
		Result []struct {
			Mid   int    `json:"mid"`
			Uname string `json:"uname"`
			Usign string `json:"usign"`
			Fans  int    `json:"fans"`
			Upic  string `json:"upic"`
		} `json:"result"`
	} `json:"data"`
}

// SearchUsers 搜索 UP 主
func SearchUsers(keyword string) ([]BiliUserMeta, error) {
	imgKey, subKey, err := GetWbiKeys()
	if err != nil {
		return nil, fmt.Errorf("failed to get wbi keys: %w", err)
	}

	params := map[string]string{
		"keyword":       keyword,
		"search_type":   "bili_user",
		"order":         "fans",
		"order_sort":    "0",
		"page":          "1",
		"pagesize":      "20",
		"platform":      "pc",
		"highlight":     "0",
		"single_column": "0",
	}

	signedParams := SignAndEncode(params, imgKey, subKey)
	apiURL := "https://api.bilibili.com/x/web-interface/wbi/search/type?" + signedParams

	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status code %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result searchUserResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	// 转换结构
	var users []BiliUserMeta
	for _, u := range result.Data.Result {
		users = append(users, BiliUserMeta{
			Mid:   u.Mid,
			Uname: u.Uname,
			Usign: u.Usign,
			Fans:  u.Fans,
			Upic:  u.Upic,
		})
	}

	return users, nil
}

// --- Wbi 实现细节 ---

func GetWbiKeys() (string, string, error) {
	cacheMutex.Lock()
	if time.Since(cacheTime) < cacheDuration && cachedImgKey != "" && cachedSubKey != "" {
		defer cacheMutex.Unlock()
		return cachedImgKey, cachedSubKey, nil
	}
	cacheMutex.Unlock()

	// 请求 nav 接口获取最新的 img_url 和 sub_url
	// 即使未登录也可以获取
	const navURL = "https://api.bilibili.com/x/web-interface/nav"
	req, _ := http.NewRequest("GET", navURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code int `json:"code"`
		Data struct {
			WbiImg struct {
				ImgUrl string `json:"img_url"`
				SubUrl string `json:"sub_url"`
			} `json:"wbi_img"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", err
	}

	imgUrl := result.Data.WbiImg.ImgUrl
	subUrl := result.Data.WbiImg.SubUrl

	imgKey := extractKey(imgUrl)
	subKey := extractKey(subUrl)

	cacheMutex.Lock()
	cachedImgKey = imgKey
	cachedSubKey = subKey
	cacheTime = time.Now()
	cacheMutex.Unlock()

	return imgKey, subKey, nil
}

func extractKey(urlStr string) string {
	parts := strings.Split(urlStr, "/")
	if len(parts) > 0 {
		filename := parts[len(parts)-1]
		return strings.Split(filename, ".")[0]
	}
	return ""
}

func getMixinKey(orig string) string {
	var sb strings.Builder
	for _, idx := range mixinKeyEncTab {
		if idx < len(orig) {
			sb.WriteByte(orig[idx])
		}
	}
	return sb.String()[:32]
}

func SignAndEncode(params map[string]string, imgKey, subKey string) string {
	mixinKey := getMixinKey(imgKey + subKey)
	currTime := strconv.FormatInt(time.Now().Unix(), 10)
	params["wts"] = currTime

	// 排序
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var queryBuilder strings.Builder
	for i, k := range keys {
		v := params[k]
		// 过滤字符
		v = strings.ReplaceAll(v, "!", "")
		v = strings.ReplaceAll(v, "'", "")
		v = strings.ReplaceAll(v, "(", "")
		v = strings.ReplaceAll(v, ")", "")
		v = strings.ReplaceAll(v, "*", "")

		if i > 0 {
			queryBuilder.WriteString("&")
		}
		queryBuilder.WriteString(url.QueryEscape(k) + "=" + url.QueryEscape(v))
	}

	query := queryBuilder.String()
	hash := md5.Sum([]byte(query + mixinKey))
	w_rid := hex.EncodeToString(hash[:])

	return query + "&w_rid=" + w_rid
}
