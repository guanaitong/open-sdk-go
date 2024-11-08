package openapi_test

import (
	"testing"

	openapi "github.com/guanaitong/open-sdk-go"
)

var sign = ""
var timestamp = int64(0)

var client *openapi.OpenClient

func init() {
	client = openapi.NewOpenClient("appId", "appSecret")
}

func TestSSOLogin(t *testing.T) {
	mobile := "17762200002"

	getAuthCodeRequest := &openapi.GetAuthCodeByMobileRequest{
		Mobile: mobile,
	}
	loginApi := &openapi.LoginApi{
		Client: client,
	}
	authCode, err := loginApi.GetAuthCodeByMobile(getAuthCodeRequest)
	if err != nil {
		t.Log(err)
	}
	request := &openapi.SSOLoginRequest{
		AuthCode:    *authCode,
		RedirectUrl: "https://m.igeidao.com",
	}
	url := loginApi.GenerateLoginUrl(request)

	t.Log(url)
}
