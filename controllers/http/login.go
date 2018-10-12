package http

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"ginder/controllers"
	"ginder/models"
)

func Login(c *gin.Context) {
	username := controllers.SetContext(c).Param("username").String()
	c.JSON(http.StatusOK, username)
}

func Register(c *gin.Context) {
	username := controllers.SetContext(c).Param("username").String()
	password := controllers.SetContext(c).Param("password").String()

	u := &models.UserLogin{Username:username, Password:password}
	existFlag := u.IsExists()

	if -1 == existFlag {
		controllers.ThrowError(c, -1, "注册失败，请稍后再试")
	} else if 1 == existFlag {
		controllers.ThrowError(c, -1, "手机号已注册")
	} else {
		c.JSON(http.StatusOK, "来吧")
	}
}
