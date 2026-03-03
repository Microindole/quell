package downloader

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var sessdata string

var (
	buvidMu      sync.Mutex
	cachedBuvid3 string
	cachedBuvid4 string
	buvidFetched bool
)

// SetSessdata 设置 SESSDATA Cookie，用于解锁高清画质
func SetSessdata(s string) {
	sessdata = s
}

// addCommonHeaders 为请求添加通用 Headers（含 Cookie）
func addCommonHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://www.bilibili.com")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	buvid3, buvid4 := fetchBuvid()
	cookie := "buvid3=" + buvid3
	if buvid4 != "" {
		cookie += "; buvid4=" + buvid4
	}
	if sessdata != "" {
		cookie += "; SESSDATA=" + sessdata
	}
	req.Header.Set("Cookie", cookie)
}

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
			cachedBuvid3 = "unknown"
			buvidFetched = true
			return cachedBuvid3, ""
		}
		b[6] = (b[6] & 0x0f) | 0x40
		b[8] = (b[8] & 0x3f) | 0x80
		cachedBuvid3 = strings.ToUpper(fmt.Sprintf("%x-%x-%x-%x-%x",
			b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])) + "infoc"
		buvidFetched = true
	}

	return cachedBuvid3, cachedBuvid4
}

func (s *DownloadState) Save(path string) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(path+".qdl", data, 0644)
}

func LoadState(path string) (*DownloadState, error) {
	data, err := os.ReadFile(path + ".qdl")
	if err != nil {
		return nil, err
	}
	var s DownloadState
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func formatSpeed(speed float64) string {
	return formatBytes(int64(speed)) + "/s"
}

// sanitizeFilename 清理文件名中的非法字符
func sanitizeFilename(name string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '\\', '/', '*', '?', '"', '<', '>', '|', ':':
			return '_'
		}
		return r
	}, name)
}
