package routers

import (
	hHome "ginder/controllers/http/home"
	//rHome "ginder/controllers/rpc/home"

	"github.com/gin-gonic/gin"
	//"github.com/hprose/hprose-golang/rpc"
)

// 过滤器
// 请求参数转换

var Router *gin.Engine

func init() {
	// route
	Router = gin.New()

	Router.Any("/home/user", hHome.PersonalInfo)

	/*Router.GET("/home/index", hHome.Home)
	Router.GET("/home/login", hHome.Login)
	Router.POST("/home/register", hHome.Register)

	v1 := Router.Group("/v1")
	{
		v1.GET("/personalInfo", hHome.PersonalInfo)
	}

	// rpc
	service := rpc.NewHTTPService()
	service.AddFunction("userInfo", rHome.UserInfo)
	Router.Any("/home", func(c *gin.Context) {
		service.ServeHTTP(c.Writer, c.Request)
	})*/

	Router.Run(":8080")
}