package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type response struct {
	Code int			`json:"code"`
	Data interface{}	`json:"data"`
	Msg  string			`json:"msg"`
}

func SwitchResponse(code int, data interface{}, msg string) *response {
	r := response{}
	r.Code = code
	r.Data = data
	r.Msg  = msg

	return &r
}

func ThrowError(c *gin.Context, code int, msg string) {
	r := SwitchResponse(code, make([]int, 0), msg)
	c.JSON(http.StatusOK, r)
}