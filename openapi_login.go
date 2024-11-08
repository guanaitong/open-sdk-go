package openapi

type GetAuthCodeByMobileRequest struct {
	Mobile string `form:"mobile"`
}

func (request GetAuthCodeByMobileRequest) IsForm() bool {
	return true
}

type SSOLoginRequest struct {
	AuthCode    string `form:"auth_code"`
	RedirectUrl string `form:"redirect_url"`
}
type LoginApi struct {
	Client *OpenClient
}

func (c *LoginApi) GetAuthCodeByMobile(request *GetAuthCodeByMobileRequest) (*string, error) {
	return Request[string](c.Client, true, "/sso/employee/getAuthCodeByMobile", request)
}

func (c *LoginApi) GenerateLoginUrl(request *SSOLoginRequest) string {
	return c.Client.buildUrl("/sso/employee/login", obj2Map(request, true))
}
