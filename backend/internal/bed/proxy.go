package bed

import (
	"crypto/aes"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
)

const housingAPI = "http://housing2021.xjtu.edu.cn"

func ProxyGet(client *resty.Client, path string, params map[string]string, token string) ([]byte, error) {
	req := client.R().
		SetHeader("Origin", "http://housing2021.xjtu.edu.cn").
		SetHeader("Referer", "http://housing2021.xjtu.edu.cn/dmWeb/")
	if token != "" {
		req.SetHeader("Token", token)
		req.SetCookie(&http.Cookie{Name: "token", Value: token, Path: "/"})
	}
	if params != nil {
		req.SetQueryParams(params)
	}
	resp, err := req.Get(housingAPI + path)
	if err != nil {
		return nil, fmt.Errorf("proxy GET %s: %w", path, err)
	}
	return resp.Body(), nil
}

func ProxyPost(client *resty.Client, path string, params map[string]string, bodyType string, token string) ([]byte, error) {
	req := client.R().
		SetHeader("Origin", "http://housing2021.xjtu.edu.cn").
		SetHeader("Referer", "http://housing2021.xjtu.edu.cn/dmWeb/")
	if token != "" {
		req.SetHeader("Token", token)
		req.SetCookie(&http.Cookie{Name: "token", Value: token, Path: "/"})
	}
	if params != nil {
		if bodyType == "query" {
			req.SetQueryParams(params)
		} else {
			req.SetFormData(params)
		}
	}
	resp, err := req.Post(housingAPI + path)
	if err != nil {
		return nil, fmt.Errorf("proxy POST %s: %w", path, err)
	}
	return resp.Body(), nil
}

// ProxyPostJSON sends a POST with JSON body to housing API (for saveBed, distributeBed, etc.)
func ProxyPostJSON(client *resty.Client, path string, body map[string]interface{}, token string) ([]byte, error) {
	req := client.R().
		SetHeader("Content-Type", "application/json; charset=UTF-8").
		SetHeader("Origin", "http://housing2021.xjtu.edu.cn").
		SetHeader("Referer", "http://housing2021.xjtu.edu.cn/dmWeb/")
	if token != "" {
		req.SetHeader("Token", token)
		req.SetCookie(&http.Cookie{Name: "token", Value: token, Path: "/"})
	}
	req.SetBody(body)
	resp, err := req.Post(housingAPI + path)
	if err != nil {
		return nil, fmt.Errorf("proxy POST JSON %s: %w", path, err)
	}
	return resp.Body(), nil
}

func EncryptBedCode(bedCode string, timestamp int64) string {
	key := []byte(fmt.Sprintf("shu%d", timestamp))
	padded := make([]byte, 16)
	copy(padded, key)
	key = padded

	block, err := aes.NewCipher(key)
	if err != nil {
		return bedCode
	}

	plaintext := []byte(bedCode)
	padLen := aes.BlockSize - len(plaintext)%aes.BlockSize
	buf := make([]byte, len(plaintext)+padLen)
	copy(buf, plaintext)
	for i := len(plaintext); i < len(buf); i++ {
		buf[i] = byte(padLen)
	}

	encrypted := make([]byte, len(buf))
	for i := 0; i < len(buf); i += aes.BlockSize {
		block.Encrypt(encrypted[i:i+aes.BlockSize], buf[i:i+aes.BlockSize])
	}

	return base64.StdEncoding.EncodeToString(encrypted)
}

// BuildDistributeBedBody 构造 distributeBed 请求体
// 原网页有两种路径：
//   choose-bed.vue 直接提交:  {personsn, bedPlaceCode(加密), divideId, aircondition, beddingInfo, chooseWay:2, t}
//   beds-collect.vue 收藏提交: 同上 + bedCodes (来自收藏项的bedCodes字段)
// 我们的抢床引擎对应收藏提交，所以 bedCodes 需要传入
func BuildDistributeBedBody(personsn, bedCode, divideId, beddingInfo, bedCodes string) map[string]interface{} {
	ts := time.Now().UnixMilli()
	body := map[string]interface{}{
		"personsn":     personsn,
		"bedPlaceCode": EncryptBedCode(bedCode, ts),
		"divideId":     divideId,
		"aircondition": "",
		"beddingInfo":  beddingInfo,
		"chooseWay":    2,
		"t":            ts,
	}
	if bedCodes != "" {
		body["bedCodes"] = bedCodes
	}
	return body
}
