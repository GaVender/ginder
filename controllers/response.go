package controllers

type response struct {
	Code int			`json:"code"`
	Data interface{}	`json:"data"`
	Msg  string			`json:"msg"`
}

func SwitchResponse(c int, d interface{}, m string) *response {
	r := response{}
	r.Code = c
	r.Data = d
	r.Msg  = m

	return &r
}
