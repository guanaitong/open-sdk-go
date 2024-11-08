package openapi

type EmployeeAddRequest struct {
	EnterpriseCode string `form:"enterpriseCode"`
	UserId         string `form:"userId"`
	Name           string `form:"name"`
	Code           string `form:"code"`
	Gender         int    `form:"gender"`
	Email          string `form:"email"`
	MobileArea     string `form:"mobileArea"`
	Mobile         string `form:"mobile"`
	SendInvite     int    `form:"sendInvite"`
	Remark         string `form:"remark"`
	DeptCode       string `form:"deptCode"`
	Level          string `form:"level"`
	BirthDay       string `form:"birthDay"`
	EntryDay       string `form:"entryDay"`
	CardType       int    `form:"cardType"`
	CardNo         string `form:"cardNo"`
	AllowSimplePwd int    `form:"allowSimplePwd"`
	Password       string `form:"password"`
}

func (request EmployeeAddRequest) IsForm() bool {
	return true
}

type EmployeeApi struct {
	Client *OpenClient
}

func (c *EmployeeApi) GetAuthCodeByMobile(request *EmployeeAddRequest) (*string, error) {
	return Request[string](c.Client, true, "/employee/add", request)
}
