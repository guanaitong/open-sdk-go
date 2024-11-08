package openapi

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	u "net/url"
	"reflect"
	"sort"
	"strings"
	"time"
)

// 生产地址
const OpenApiProdUrl = "https://openapi.guanaitong.com"

// 测试地址
const OpenApiTestUrl = "https://openapi.guanaitong.tech"

const grantType = "client_credential"

type Code int

const (
	CodeOk Code = iota
)

type StatusError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (e *StatusError) Error() string {
	return e.Msg
}

type ApiResponse[T any] struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data"`
}

type ApiRequest interface {
	IsForm() bool
}

type CreateTokenRequest struct {
	GrantType string `form:"grant_type"`
}

func (request CreateTokenRequest) IsForm() bool {
	return true
}

type OpenClient struct {
	BaseUrl    string
	AppId      string
	AppSecret  string
	HttpClient *http.Client
	Token      *Token
}

func NewOpenClient(appId string, appSecret string, isProd bool) *OpenClient {
	c := &OpenClient{
		AppId:      appId,
		AppSecret:  appSecret,
		HttpClient: &http.Client{},
	}
	if isProd {
		c.BaseUrl = OpenApiProdUrl
	} else {
		c.BaseUrl = OpenApiTestUrl

	}
	return c
}

type Token struct {
	AccessToken string
	ExpiresIn   int64
	CreateAt    int64
	ExpiresAt   int64
}

type TokenResp struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

func newToken(tokenResp *TokenResp) *Token {
	timeStamp := time.Now().UnixMilli()
	return &Token{
		AccessToken: tokenResp.AccessToken,
		ExpiresIn:   tokenResp.ExpiresIn,
		CreateAt:    timeStamp,
		ExpiresAt:   timeStamp + tokenResp.ExpiresIn*1000,
	}
}

func (token *Token) isExpired() bool {
	return time.Now().UnixMilli() < token.ExpiresAt
}

func (token *Token) needRefresh() bool {
	// 获取当前时间的Unix时间戳，单位为秒
	currentTime := time.Now().UnixMilli()
	return (currentTime - token.CreateAt) > (800 * token.ExpiresIn)
}

func (c *OpenClient) refreshToken() {
	c.Token = nil
	c.GetToken()
}

func (c *OpenClient) buildUrl(path string, params map[string]any) string {
	url, _ := url.JoinPath(c.BaseUrl, path)
	if len(params) > 0 {
		return url + "?" + c.buildQuery(params)
	} else {
		return url
	}
}

func (c *OpenClient) buildQuery(params map[string]any) string {
	var values = make(u.Values)
	for key, value := range params {
		values.Set(key, fmt.Sprint(value))
	}
	return values.Encode()
}

func doPost[T any](url string, body, contentType string) (*ApiResponse[T], error) {
	// params
	var bodyData io.Reader
	if body == "" {
		bodyData = nil
	} else {
		bodyData = strings.NewReader(body)
	}
	req, _ := http.NewRequest("POST", url, bodyData)
	req.Header.Set("Content-Type", contentType)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("remote error: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failure,http status is %d", resp.StatusCode)
	}
	if data, err := io.ReadAll(resp.Body); err != nil {
		return nil, fmt.Errorf("read response error %w", err)
	} else {
		var value ApiResponse[T]
		if err := json.Unmarshal(data, &value); err != nil {
			return nil, fmt.Errorf("failed parse apiresponse error %w", err)
		} else {
			return &value, nil
		}
	}
}

func obj2Map(obj any, isForm bool) map[string]any {
	resultMap := make(map[string]any)
	objValues := reflect.ValueOf(obj)

	var types reflect.Type
	if objValues.Kind() == reflect.Ptr {
		objValues = objValues.Elem() // 解引用指针
		types = objValues.Type()
	} else {
		types = objValues.Type()
	}

	if isForm {
		for i := 0; i < types.NumField(); i++ {
			field := types.Field(i)
			if tag := field.Tag.Get("form"); tag != "" {
				resultMap[tag] = objValues.Field(i).Interface()
			}
		}
	} else {
		d, _ := json.Marshal(obj)

		resultMap["_body"] = string(d)
	}
	return resultMap
}

func Request[T any](c *OpenClient, auth bool, path string, request ApiRequest) (*T, error) {
	timeStamp := time.Now().Unix()
	commonParams := map[string]any{
		"appid":     c.AppId,
		"timestamp": timeStamp,
	}
	if auth {
		err := c.GetToken()
		if err != nil {
			return nil, err
		}
		commonParams["access_token"] = c.Token.AccessToken
	}
	bizParams := obj2Map(request, request.IsForm())
	sign := c.sign(commonParams, bizParams)

	commonParams["sign"] = sign

	contentType, bodyVal := "", ""
	if request.IsForm() {
		contentType = "application/x-www-form-urlencoded"
		bodyVal = c.buildQuery(bizParams)
	} else {
		contentType = "application/json"
		if _, ok := bizParams["_body"]; ok {
			bodyVal = bizParams["_body"].(string)
		}
	}
	url := c.buildUrl(path, commonParams)
	apiResp, err := doPost[T](url, bodyVal, contentType)
	if err != nil {
		return nil, err
	}
	if apiResp.Code == 0 {
		return &apiResp.Data, nil
	} else if apiResp.Code == 1000210004 {
		// 如果HTTP请求后，token已失效，默认清空，重新生成新的token
		c.refreshToken()
		return Request[T](c, auth, path, request)
	} else {
		return nil, &StatusError{
			Code: apiResp.Code,
			Msg:  apiResp.Msg,
		}
	}
}

func (client *OpenClient) sign(commonParams map[string]any, bizParams map[string]any) string {
	toSignParams := make(map[string]any)
	toSignParams["appsecret"] = client.AppSecret
	for k, v := range commonParams {
		toSignParams[k] = v
	}
	for k, v := range bizParams {
		toSignParams[k] = v
	}
	var paramsList []string
	for key, value := range toSignParams {
		if key != "sign" {
			paramsList = append(paramsList, fmt.Sprintf("%s=%v", key, value))
		}
	}
	// Sort the parameters to ensure consistent order
	sort.Strings(paramsList)
	// Join the parameters with "&"
	result := strings.Join(paramsList, "&")
	// 计算SHA1哈希值
	hash := sha1.New()
	hash.Write([]byte(result))
	sha1Bytes := hash.Sum(nil)

	// 将哈希值转换为十六进制字符串
	sign := hex.EncodeToString(sha1Bytes)
	return sign
}

func (c *OpenClient) GetToken() error {
	if c.Token == nil || c.Token.needRefresh() {
		t, err := c.createToken()
		if err != nil {
			return err
		}
		c.Token = t
		return nil
	}
	return nil
}

func (c *OpenClient) createToken() (*Token, error) {
	tokenResp, err := Request[TokenResp](c, false, "/token/create", &CreateTokenRequest{
		GrantType: grantType,
	})
	if err != nil {
		return nil, fmt.Errorf("createToken error,%w", err)
	}
	return newToken(tokenResp), nil
}
