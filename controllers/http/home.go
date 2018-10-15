package http

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"ginder/controllers"
	"fmt"
	"time"
)

type HomeOutput struct {
	Title string `json:"title"`
	NowTime  string `json:"now_time"`
}

func Home(c *gin.Context) {
	token := controllers.SetContext(c).Param("token").String()
	userInfo := GetUserInfoRedisDataByToken(token)

	if userInfo.Username == "" {
		controllers.ThrowError(c, -1, "登录超时，请重新登录")
	} else {
		output := HomeOutput{}
		output.Title = fmt.Sprintf("欢迎您，%s", userInfo.Username)
		output.NowTime = time.Now().Format("2006-01-02 15:04:05")
		c.JSON(http.StatusOK, controllers.SwitchResponse(0, output, ""))
	}
}