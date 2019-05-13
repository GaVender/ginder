package routers

import (
	"fmt"

	"github.com/GaVender/ginder/conf"
	"github.com/GaVender/ginder/controllers/http/home"

	"github.com/gin-gonic/gin"
	"github.com/hprose/hprose-golang/rpc"
)

// 过滤器
// 请求参数转换

const Address = ":8081"

var Router *gin.Engine

func init() {
	// environment
	conf.GetMasterMysql()
	defer conf.CloseMasterMysql()

	conf.GetSlaveMysql()
	defer conf.CloseSlaveMysql()

	conf.GetMasterRedis()
	defer conf.CloseMasterRedis()

	conf.GetSlaveRedis()
	defer conf.CloseSlaveRedis()

	defer conf.CloseMongoSession()

	// route
	gin.SetMode(gin.DebugMode)
	Router = gin.New()

	Router.Any("/home/user", home.PersonalInfo)

	Router.GET("/home/index", home.Home)
	Router.GET("/home/login", home.Login)
	Router.POST("/home/register", home.Register)

	v1 := Router.Group("/v1")
	{
		v1.GET("/personalInfo", home.PersonalInfo)
	}

	// rpc
	service := rpc.NewHTTPService()
	service.AddFunction("userInfo", home.PersonalInfo)
	Router.Any("/home", func(c *gin.Context) {
		service.ServeHTTP(c.Writer, c.Request)
	})

	if err := Router.Run(Address); err != nil {
		conf.GetErrorLogger().LogError("http server", fmt.Sprintf("%s start error: %s", Address, err.Error()))
		panic("http server start error: " + err.Error())
	}
}