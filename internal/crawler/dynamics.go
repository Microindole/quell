package crawler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

func normalizeURL(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if strings.HasPrefix(s, "//") {
		return "https:" + s
	}
	return s
}

func getUserDynamicsOnce(mid, offset string, withSess bool) (*DynamicListResult, int, string, error) {
	uid := strings.TrimSpace(mid)
	if uid == "" {
		return nil, -1, "", fmt.Errorf("UID 不能为空")
	}

	params := url.Values{}
	params.Set("host_mid", uid)
	if strings.TrimSpace(offset) != "" {
		params.Set("offset", offset)
	}

	apiURL := "https://api.bilibili.com/x/polymer/web-dynamic/v1/feed/space?" + params.Encode()
	req, _ := http.NewRequest("GET", apiURL, nil)
	addCommonHeadersWithSess(req, withSess)
	req.Header.Set("Referer", "https://space.bilibili.com/"+uid+"/dynamic")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, -1, "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			HasMore bool   `json:"has_more"`
			Offset  string `json:"offset"`
			Items   []struct {
				IDStr   string `json:"id_str"`
				Type    string `json:"type"`
				Modules struct {
					ModuleAuthor struct {
						Name    string      `json:"name"`
						Face    string      `json:"face"`
						PubTime string      `json:"pub_time"`
						PubTs   interface{} `json:"pub_ts"`
					} `json:"module_author"`
					ModuleDynamic struct {
						Desc struct {
							Text          string `json:"text"`
							RichTextNodes []struct {
								Type    string `json:"type"`
								Text    string `json:"text"`
								JumpURL string `json:"jump_url"`
								Emoji   struct {
									IconURL string `json:"icon_url"`
									Text    string `json:"text"`
								} `json:"emoji"`
							} `json:"rich_text_nodes"`
						} `json:"desc"`
						Major struct {
							Type string `json:"type"`
							Draw struct {
								Items []struct {
									Src string `json:"src"`
								} `json:"items"`
							} `json:"draw"`
							Archive struct {
								Bvid  string `json:"bvid"`
								Title string `json:"title"`
								Cover string `json:"cover"`
							} `json:"archive"`
						} `json:"major"`
					} `json:"module_dynamic"`
				} `json:"modules"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, -1, "", fmt.Errorf("解析动态列表失败: %w", err)
	}
	if result.Code != 0 {
		return nil, result.Code, result.Message, nil
	}

	items := make([]BiliDynamicItem, 0, len(result.Data.Items))
	for _, it := range result.Data.Items {
		d := BiliDynamicItem{
			IDStr:      it.IDStr,
			Type:       it.Type,
			UserName:   it.Modules.ModuleAuthor.Name,
			UserFace:   normalizeURL(it.Modules.ModuleAuthor.Face),
			PubTime:    it.Modules.ModuleAuthor.PubTime,
			PubTs:      parseInt64Any(it.Modules.ModuleAuthor.PubTs),
			Text:       it.Modules.ModuleDynamic.Desc.Text,
			Bvid:       it.Modules.ModuleDynamic.Major.Archive.Bvid,
			VideoTitle: it.Modules.ModuleDynamic.Major.Archive.Title,
			VideoCover: normalizeURL(it.Modules.ModuleDynamic.Major.Archive.Cover),
		}
		for _, n := range it.Modules.ModuleDynamic.Desc.RichTextNodes {
			nodeText := n.Text
			emojiURL := normalizeURL(n.Emoji.IconURL)
			if nodeText == "" {
				nodeText = n.Emoji.Text
			}
			d.RichNodes = append(d.RichNodes, DynamicRichNode{
				Type:     n.Type,
				Text:     nodeText,
				JumpURL:  normalizeURL(n.JumpURL),
				EmojiURL: emojiURL,
			})
		}
		if d.Text == "" {
			d.Text = "(无文字内容)"
		}
		for _, img := range it.Modules.ModuleDynamic.Major.Draw.Items {
			if u := normalizeURL(img.Src); u != "" {
				d.ImageURLs = append(d.ImageURLs, u)
			}
		}
		items = append(items, d)
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].PubTs > items[j].PubTs })

	return &DynamicListResult{
		Items:   items,
		Offset:  result.Data.Offset,
		HasMore: result.Data.HasMore,
	}, 0, "", nil
}

// GetUserDynamics 获取指定 UP 的动态流（空间动态）。
func GetUserDynamics(mid, offset string) (*DynamicListResult, error) {
	res, code, msg, err := getUserDynamicsOnce(mid, offset, true)
	if err != nil {
		return nil, err
	}
	if code == 0 {
		return res, nil
	}
	if code == -412 {
		time.Sleep(1200 * time.Millisecond)
		res2, code2, msg2, err2 := getUserDynamicsOnce(mid, offset, false)
		if err2 != nil {
			return nil, err2
		}
		if code2 == 0 {
			return res2, nil
		}
		if code2 == -412 {
			time.Sleep(1500 * time.Millisecond)
			res3, code3, msg3, err3 := getUserDynamicsOnce(mid, offset, true)
			if err3 != nil {
				return nil, err3
			}
			if code3 == 0 {
				return res3, nil
			}
			return nil, fmt.Errorf("b站 API 错误 (code=%d): %s（已重试两次）", code3, msg3)
		}
		return nil, fmt.Errorf("b站 API 错误 (code=%d): %s（已重试一次）", code2, msg2)
	}
	return nil, fmt.Errorf("b站 API 错误 (code=%d): %s", code, msg)
}
