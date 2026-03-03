package crawler

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var mixinKeyEncTab = []int{
	46, 47, 18, 2, 53, 8, 23, 32, 15, 50, 10, 31, 58, 3, 45, 35, 27, 43, 5, 49,
	33, 9, 42, 19, 29, 28, 14, 39, 12, 38, 41, 13, 37, 48, 7, 16, 24, 55, 40,
	61, 26, 17, 0, 1, 60, 51, 30, 4, 22, 25, 54, 21, 56, 59, 6, 63, 57, 62, 11,
	36, 20, 34, 44, 52,
}

var (
	cacheMutex    sync.Mutex
	cachedImgKey  string
	cachedSubKey  string
	cacheTime     time.Time
	cacheDuration = 10 * time.Minute
)

func resetWbiCache() {
	cacheMutex.Lock()
	cachedImgKey = ""
	cachedSubKey = ""
	cacheTime = time.Time{}
	cacheMutex.Unlock()
}

func GetWbiKeys() (string, string, error) {
	cacheMutex.Lock()
	if time.Since(cacheTime) < cacheDuration && cachedImgKey != "" && cachedSubKey != "" {
		defer cacheMutex.Unlock()
		return cachedImgKey, cachedSubKey, nil
	}
	cacheMutex.Unlock()

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
				ImgURL string `json:"img_url"`
				SubURL string `json:"sub_url"`
			} `json:"wbi_img"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", err
	}

	imgKey := extractKey(result.Data.WbiImg.ImgURL)
	subKey := extractKey(result.Data.WbiImg.SubURL)

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
	params["wts"] = strconv.FormatInt(time.Now().Unix(), 10)

	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var queryBuilder strings.Builder
	for i, k := range keys {
		v := params[k]
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
	wRid := hex.EncodeToString(hash[:])
	return query + "&w_rid=" + wRid
}
