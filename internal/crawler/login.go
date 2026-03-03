package crawler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type webLoginGenerateResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		URL       string `json:"url"`
		QrcodeKey string `json:"qrcode_key"`
	} `json:"data"`
}

type webLoginPollResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Code int    `json:"code"`
		URL  string `json:"url"`
	} `json:"data"`
}

// GenerateWebLoginQRCode 生成网页登录二维码地址和 qrcode_key。
func GenerateWebLoginQRCode() (loginURL, qrcodeKey string, err error) {
	api := "https://passport.bilibili.com/x/passport-login/web/qrcode/generate?source=main-fe-header"
	req, _ := http.NewRequest("GET", api, nil)
	addCommonHeaders(req)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("请求登录二维码失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result webLoginGenerateResp
	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", fmt.Errorf("解析登录二维码响应失败: %w", err)
	}
	if result.Code != 0 {
		return "", "", fmt.Errorf("B站登录接口错误 (code=%d): %s", result.Code, result.Message)
	}
	if result.Data.URL == "" || result.Data.QrcodeKey == "" {
		return "", "", fmt.Errorf("登录二维码响应缺少必要字段")
	}

	return result.Data.URL, result.Data.QrcodeKey, nil
}

// PollWebLoginStatus 轮询二维码登录状态。
// 返回 status: waiting_scan / waiting_confirm / expired / success
func PollWebLoginStatus(qrcodeKey string) (status, message, sess string, err error) {
	api := "https://passport.bilibili.com/x/passport-login/web/qrcode/poll?qrcode_key=" + url.QueryEscape(qrcodeKey) + "&source=main-fe-header"
	req, _ := http.NewRequest("GET", api, nil)
	addCommonHeaders(req)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", "", fmt.Errorf("轮询登录状态失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result webLoginPollResp
	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", "", fmt.Errorf("解析登录状态失败: %w", err)
	}
	if result.Code != 0 {
		return "", "", "", fmt.Errorf("B站登录状态接口错误 (code=%d): %s", result.Code, result.Message)
	}

	switch result.Data.Code {
	case 86101:
		return "waiting_scan", "等待扫码", "", nil
	case 86090:
		return "waiting_confirm", "已扫码，等待手机确认", "", nil
	case 86038:
		return "expired", "二维码已过期", "", nil
	case 0:
		sessVal, err := extractSessdataFromURL(result.Data.URL)
		if err != nil {
			return "", "", "", err
		}
		return "success", "登录成功", sessVal, nil
	default:
		return "", "", "", fmt.Errorf("未知登录状态码: %d", result.Data.Code)
	}
}

func extractSessdataFromURL(raw string) (string, error) {
	if raw == "" {
		return "", fmt.Errorf("登录成功但返回 URL 为空")
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("解析登录返回 URL 失败: %w", err)
	}
	q := u.Query()
	sess := q.Get("SESSDATA")
	if sess == "" {
		return "", fmt.Errorf("登录成功但未获取到 SESSDATA")
	}
	return sess, nil
}
