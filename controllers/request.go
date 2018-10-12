package controllers

import (
	"github.com/gin-gonic/gin"
	"strconv"
)

type Input struct {
	context *gin.Context
}

type Body struct {
	body string
}

func SetContext(c *gin.Context) *Input {
	i := &Input{context:c}
	return i
}

func (i *Input) Param(name string) *Body {
	c := i.context
	b := &Body{}
	ret := ""

	if "GET" == c.Request.Method {
		ret = c.Query(name)
	} else if "POST" == c.Request.Method {
		ret = c.PostForm(name)
	}

	b.body = ret
	return b
}

func (b *Body) Int() int {
	ret, _ := strconv.Atoi(b.body)
	return ret
}

func (b *Body) String() string {
	return b.body
}