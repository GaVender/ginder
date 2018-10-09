package http

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"ginder/controllers"
	"ginder/conf"
	"ginder/models"
)

func Home(c *gin.Context) {
	dbSlave := conf.SqlSlaveDb()
	defer dbSlave.Close()

	var u models.User
	err := dbSlave.Get(&u, "select * from passport.user where username = ?", "13631277247")

	if err != nil {
		conf.LogicLogger.Error("mysql select error: %s", err.Error())
	}

	r := controllers.SwitchResponse(0, u, "")
	c.JSON(http.StatusOK, r)
}
