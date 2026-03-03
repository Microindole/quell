package crawler

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

var sessdata string

// SetSessdata 设置 SESSDATA Cookie。
func SetSessdata(s string) {
	sessdata = s
}

var (
	buvidMu      sync.Mutex
	cachedBuvid3 string
	cachedBuvid4 string
	buvidFetched bool
)

func fetchBuvid() (string, string) {
	buvidMu.Lock()
	defer buvidMu.Unlock()

	if buvidFetched {
		return cachedBuvid3, cachedBuvid4
	}

	type spiResp struct {
		Code int `json:"code"`
		Data struct {
			B3 string `json:"b_3"`
			B4 string `json:"b_4"`
		} `json:"data"`
	}

	req, _ := http.NewRequest("GET", "https://api.bilibili.com/x/frontend/finger/spi", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err == nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		var result spiResp
		if json.Unmarshal(body, &result) == nil && result.Code == 0 {
			cachedBuvid3 = result.Data.B3
			cachedBuvid4 = result.Data.B4
			buvidFetched = true
		}
	}

	if cachedBuvid3 == "" {
		b := make([]byte, 16)
		if _, err := rand.Read(b); err != nil {
			return "", ""
		}
		b[6] = (b[6] & 0x0f) | 0x40
		b[8] = (b[8] & 0x3f) | 0x80
		cachedBuvid3 = strings.ToUpper(fmt.Sprintf("%x-%x-%x-%x-%x",
			b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])) + "infoc"
		buvidFetched = true
	}

	return cachedBuvid3, cachedBuvid4
}

func addCommonHeaders(req *http.Request) {
	addCommonHeadersWithSess(req, true)
}

func addCommonHeadersWithSess(req *http.Request, withSess bool) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Referer", "https://www.bilibili.com")

	buvid3, buvid4 := fetchBuvid()
	cookie := "buvid3=" + buvid3
	if buvid4 != "" {
		cookie += "; buvid4=" + buvid4
	}
	if withSess && sessdata != "" {
		cookie += "; SESSDATA=" + sessdata
	}
	req.Header.Set("Cookie", cookie)
}
