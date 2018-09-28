package http

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func PersonalInfo (c *gin.Context) {
	c.String(http.StatusOK, "hello man")
}
