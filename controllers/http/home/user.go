package home

import (
	"net/http"

	"github.com/GaVender/ginder/controllers"

	"github.com/gin-gonic/gin"
)

func PersonalInfo (c *gin.Context) {
	controllers.ThrowError(c, -1, "首页显示异常，请稍后再试")
	c.JSON(http.StatusOK, controllers.SwitchResponse(0, 123, ""))
}
