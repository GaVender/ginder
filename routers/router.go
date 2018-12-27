package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/hprose/hprose-golang/rpc"

	h_home "ginder/controllers/http/home"
	r_home "ginder/controllers/rpc/home"
)

// 过滤器
// 请求参数转换

var Router *gin.Engine

func init() {
	// route
	Router = gin.New()

	Router.GET("/home/index", h_home.Home)
	Router.GET("/home/login", h_home.Login)
	Router.POST("/home/register", h_home.Register)

	v1 := Router.Group("/v1")
	{
		v1.GET("/personalInfo", h_home.PersonalInfo)
	}

	// rpc
	service := rpc.NewHTTPService()
	service.AddFunction("userInfo", r_home.UserInfo)
	Router.Any("/home", func(c *gin.Context) {
		service.ServeHTTP(c.Writer, c.Request)
	})

	Router.Run(":8080")
}