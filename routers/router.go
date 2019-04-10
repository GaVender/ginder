package routers

import (
	"github.com/GaVender/ginder/controllers/http/monitor"
	"github.com/GaVender/ginder/command/sms"

	"github.com/xiaobai22/gokit-service/monitorkit"
	"github.com/gin-gonic/gin"
)

// 过滤器
// 请求参数转换

var Router *gin.Engine

func init() {
	start()
}

func start() {
	// route
	/*Router = gin.New()

	Router.Any("/home/user", hHome.PersonalInfo)

	Router.GET("/home/index", hHome.Home)
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
	})

	Router.Run(":8080")*/

	go monitorkit.StartMonitorBB("9091", "/black")
	go sms.Monitor()
	go sms.SendProcedure(sms.SmsTypeMw)
	go sms.SendProcedure(sms.SmsTypeWl)

	gin.SetMode(gin.DebugMode)
	Router = gin.New()
	Router.GET("/monitor", monitor.Sms)
	Router.Run(":8081")
}