package route

import (
	"github.com/gin-gonic/gin"
	"ginder/api/http"
)

// 过滤器
// 请求参数转换

var Router *gin.Engine

func init() {
	Router = gin.New()

	Router.GET("/home/index", http.Home)
	Router.GET("/home/login", http.Login)
	Router.POST("/home/register", http.Register)

	v1 := Router.Group("/v1")
	{
		v1.GET("/personalInfo", http.PersonalInfo)
	}
}
