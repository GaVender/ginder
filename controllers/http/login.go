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
	TOKEN_USER_INFO_REDIS 	 = "string:token:user_info:"
	USERNAME_USER_INFO_REDIS = "string:username:user_info:"
	USER_INFO_REDIS_TIME 	 = time.Second * 60 * 60 * 24 * 30
)

type UserInfoOutput struct {
	Token 	 string `json:"token"`
	Username string `json:"username"`
}

func Login(c *gin.Context) {
	username := controllers.SetContext(c).Param("username").String()
	password := controllers.SetContext(c).Param("password").String()

	u := &models.UserLogin{Username: username, Password: password}
	existFlag := u.IsExists()

	if -1 == existFlag {
		controllers.ThrowError(c, -1, "登录失败，请稍后再试")
	} else if 1 == existFlag {
		errorFlag := true
		userInfoRedis := GetUserInfoRedisDataByUsername(u.Username)

		if userInfoRedis.Username == "" {
			user := u.UserInfo()

			if user.Id == 0  {
				controllers.ThrowError(c, -1, "手机号未注册")
			} else {
				if user.Password != u.EncryptPassword() {
					controllers.ThrowError(c, -1, "登录密码错误，请重新登录")
				} else {
					token := u.CreateUserToken()
					userInfo := createUserInfoRedisData(token, user.Id, u)
					saveUserInfoRedis(token, u.Username, userInfo)
					errorFlag = false
				}
			}
		} else {
			if userInfoRedis.Password != u.EncryptPassword() {
				controllers.ThrowError(c, -1, "登录密码错误，请重新登录")
			} else {
				errorFlag = false
			}
		}

		if !errorFlag {
			userInfoRedis = GetUserInfoRedisDataByUsername(u.Username)

			if userInfoRedis.Username == "" {
				controllers.ThrowError(c, -1, "登录失败，请稍后再试")
			} else {
				userInfoOutput := createUserInfoOutputData(userInfoRedis)
				c.JSON(http.StatusOK, controllers.SwitchResponse(0, *userInfoOutput, ""))
			}
		}
	} else {
		controllers.ThrowError(c, -1, "手机号未注册")
	}
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
				conf.LoggerLogic().Error("user register error : %s", err.Error())
				controllers.ThrowError(c, -1, "注册成功，请登录")
			} else {
				token := u.CreateUserToken()
				userInfo := createUserInfoRedisData(token, id, u)
				saveUserInfoRedis(token, u.Username, userInfo)

				userInfoRedis := GetUserInfoRedisDataByUsername(u.Username)
				userInfoOutput := createUserInfoOutputData(userInfoRedis)
				c.JSON(http.StatusOK, controllers.SwitchResponse(0, *userInfoOutput, ""))
			}
		}
	}
}

func GetUserInfoRedisDataByUsername(username string) *models.UserInfoRedis {
	u := models.UserInfoRedis{}
	redis := conf.RedisSlave()
	ret := redis.Get(USERNAME_USER_INFO_REDIS + username)

	if ret.Err() != nil {
		conf.LoggerLogic().Error("手机号：%s 没有用户信息redis，error：%s", username, ret.Err().Error())
	} else {
		json.Unmarshal([]byte(ret.Val()), &u)
	}

	return &u
}

func GetUserInfoRedisDataByToken(token string) *models.UserInfoRedis {
	u := models.UserInfoRedis{}
	redis := conf.RedisSlave()
	ret := redis.Get(TOKEN_USER_INFO_REDIS + token)

	if ret.Err() == nil {
		json.Unmarshal([]byte(ret.Val()), &u)
	}

	return &u
}



func createUserInfoRedisData(token string, uid int64, u *models.UserLogin) *models.UserInfoRedis{
	user := models.UserInfoRedis{}
	user.Token = token
	user.Uid = uid
	user.Username = u.Username
	user.Password = u.EncryptPassword()
	return &user
}

func saveUserInfoRedis(token string, username string, u *models.UserInfoRedis) {
	uJson, _ := json.Marshal(*u)
	redis := conf.RedisMaster()
	redis.Set(TOKEN_USER_INFO_REDIS + token, string(uJson), USER_INFO_REDIS_TIME)
	redis.Set(USERNAME_USER_INFO_REDIS + username, string(uJson), USER_INFO_REDIS_TIME)
}

func createUserInfoOutputData(u *models.UserInfoRedis) *UserInfoOutput {
	return &UserInfoOutput{
		Token: u.Token,
		Username: u.Username,
	}
}
