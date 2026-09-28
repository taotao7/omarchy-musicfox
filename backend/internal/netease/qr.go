package netease

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-musicfox/netease-music/util"
	"github.com/imroc/req/v3"
)

const qrLoginUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

type qrResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	UniKey  string `json:"unikey"`
}

func qrGetKey(jar http.CookieJar) (string, string, error) {
	form, err := util.ApiParamsEncode(map[string]any{
		"type":         1,
		"noCheckToken": true,
	})
	if err != nil {
		return "", "", fmt.Errorf("encode QR key request: %w", err)
	}

	response, err := newQRClient(nil).R().
		SetHeaders(map[string]string{
			"Referer":            "https://music.163.com/",
			"Origin":             "https://music.163.com",
			"Accept-Language":    "zh-CN,zh;q=0.9,en;q=0.8",
			"Cache-Control":      "no-cache",
			"Pragma":             "no-cache",
			"Sec-Ch-Ua":          `"Chromium";v="124", "Google Chrome";v="124", "Not-A.Brand";v="99"`,
			"Sec-Ch-Ua-Mobile":   "?0",
			"Sec-Ch-Ua-Platform": `"macOS"`,
			"Sec-Fetch-Dest":     "empty",
			"Sec-Fetch-Mode":     "cors",
			"Sec-Fetch-Site":     "same-origin",
			"X-Channelsource":    "undefined",
			"X-Os":               "web",
		}).
		SetFormData(form).
		Post("https://music.163.com/weapi/login/qrcode/unikey")
	if err != nil {
		return "", "", fmt.Errorf("send QR key request: %w", err)
	}

	payload, err := decodeQRResponse(response.Bytes())
	if err != nil {
		return "", "", fmt.Errorf("decode QR key response: %w", err)
	}
	if payload.Code != 200 || payload.UniKey == "" {
		return "", "", fmt.Errorf("QR key request failed (%d): %s", payload.Code, compact(response.Bytes()))
	}

	loginURL := "http://music.163.com/login?codekey=" + payload.UniKey + "&chainId=" + util.GenerateChainID(jar)
	return payload.UniKey, loginURL, nil
}

func qrCheck(key string, jar http.CookieJar) (qrResponse, error) {
	util.ApplyRequestStrategy(jar)
	form, err := util.ApiParamsEncode(map[string]any{
		"type":         1,
		"noCheckToken": true,
		"key":          key,
	})
	if err != nil {
		return qrResponse{}, fmt.Errorf("encode QR status request: %w", err)
	}

	response, err := newQRClient(jar).R().
		SetHeaders(map[string]string{
			"Referer": "https://music.163.com/",
			"Origin":  "https://music.163.com",
		}).
		SetFormData(form).
		Post("https://music.163.com/weapi/login/qrcode/client/login")
	if err != nil {
		return qrResponse{}, fmt.Errorf("send QR status request: %w", err)
	}
	payload, err := decodeQRResponse(response.Bytes())
	if err != nil {
		return qrResponse{}, fmt.Errorf("decode QR status response: %w", err)
	}
	return payload, nil
}

func newQRClient(jar http.CookieJar) *req.Client {
	return req.C().
		SetTimeout(15 * time.Second).
		SetUserAgent(qrLoginUserAgent).
		SetCookieJar(jar).
		SetTLSFingerprintChrome()
}

func decodeQRResponse(body []byte) (qrResponse, error) {
	var response qrResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return qrResponse{}, err
	}
	return response, nil
}
