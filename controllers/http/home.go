package http

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"ginder/controllers"
	"ginder/conf"
	"ginder/models"
	"encoding/json"
	"fmt"
)

func Home(c *gin.Context) {
	var u models.User
	phone := controllers.SetContext(c).Param("username").String()
	userInfoKey := "hash:user_info"

	redis := conf.RedisMaster()

	if !redis.Exists(userInfoKey).Val() || !redis.HExists(userInfoKey, phone).Val() {
		dbSlave := conf.SqlSlaveDb()
		defer dbSlave.Close()

		err := dbSlave.Get(&u, "select * from passport.user where username = ?", phone)

		if err != nil {
			conf.LoggerLogic().Error("mysql select error: %s", err.Error())
			controllers.ThrowError(c, -1, err.Error())
		} else {
			r, _ := json.Marshal(u)
			ret := redis.HSet(userInfoKey, u.Username, string(r))

			if !ret.Val() {
				conf.LoggerLogic().Error("redis save user info : %s, error: %s", phone, ret.String())
				controllers.ThrowError(c, -1, ret.String())
			}
		}
	}

	rJson := redis.HGet(userInfoKey, phone).Val()

	if rJson != "" {
		err := json.Unmarshal([]byte(rJson), &u)

		if err != nil {
			conf.LoggerLogic().Error("user info json unmarsha1 error: %s", err.Error())
			controllers.ThrowError(c, -1, err.Error())
		} else {
			r := controllers.SwitchResponse(0, u, "")
			c.JSON(http.StatusOK, r)
		}
	} else {
		errMsg := fmt.Sprintf("user redis info : %s, not exist", phone)
		conf.LoggerLogic().Error(errMsg)
		controllers.ThrowError(c, -1, errMsg)
	}
}