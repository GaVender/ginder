package http

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"ginder/controllers"
	"ginder/models"
	"ginder/conf"
	"encoding/json"
	"time"
)

const (
	TOKEN_USER__INFO_REDIS 	 = "token:user_info:"
	USERNAME_USER_INFO_REDIS = "username:user_info:"
	USER_INFO_REDIS_TIME 	 = time.Second * 60 * 60 * 24
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
		ret, err := u.Register()

		if err != nil {
			conf.LoggerLogic().Error("user register error : %s", err.Error())
			controllers.ThrowError(c, -1, "注册出错，请稍后再试")
		} else {
			id, err := ret.LastInsertId()

			if err != nil {

			} else {
				token := u.CreateUserToken()
				userInfo := models.UserInfoRedis{
					Token: token,
					Uid: id,
					Username: u.Username,
					Password: u.EncryptPassword(),
				}

				uJson, _ := json.Marshal(userInfo)
				redis := conf.RedisMaster()
				redis.SetNX(TOKEN_USER__INFO_REDIS + token, uJson, USER_INFO_REDIS_TIME)
				redis.SetNX(USERNAME_USER_INFO_REDIS + u.Username, uJson, USER_INFO_REDIS_TIME)
			}

			c.JSON(http.StatusOK, "注册成功")
		}
	}
}
