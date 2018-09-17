package http

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"ginder/api"
)

type home struct {
	Title string
	Time string
}

func Home(c *gin.Context) {
	d := home{Title: "Welcome", Time: "2017"}
	r := api.SwitchResponse(0, d, "")
	c.JSON(http.StatusOK, r)
}
