package http

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func Login(c *gin.Context) {
	c.String(http.StatusOK, "login")
}

func Register(c *gin.Context) {
	c.String(http.StatusOK, "register")
}
