package crawler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// SearchUsers 搜索 UP 主。
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
	apiURL := "https://api.bilibili.com/x/web-interface/wbi/search/type?" + SignAndEncode(params, imgKey, subKey)

	req, _ := http.NewRequest("GET", apiURL, nil)
	addCommonHeaders(req)
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
	if result.Code != 0 {
		return nil, fmt.Errorf("b站 API 错误 (code=%d): %s", result.Code, result.Message)
	}

	users := make([]BiliUserMeta, 0, len(result.Data.Result))
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
