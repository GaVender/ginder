package home

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"ginder/controllers"
	"fmt"
	"time"
	"gopkg.in/mgo.v2/bson"
	"ginder/conf"
	"gopkg.in/mgo.v2"
)

type HomeOutput struct {
	ID       bson.ObjectId `json:"id" bson:"_id"`
	Title 	 string `json:"title" bson:"title"`
	Username string `json:"username" bson:"username"`
	NowTime	 string `json:"now_time" bson:"now_time"`
}

func Home(c *gin.Context) {
	token := controllers.SetContext(c).Param("token").String()
	userInfo := GetUserInfoRedisDataByToken(token)

	if userInfo.Username == "" {
		controllers.ThrowError(c, -1, "登录超时，请重新登录")
	} else {
		mongoSession := conf.MongoSession()
		defer mongoSession.Close()

		con := mongoSession.DB("passport").C("home_info")
		output := HomeOutput{}

		if err := con.Find(bson.M{"username": userInfo.Username}).One(&output); err != nil {
			if err.Error() != mgo.ErrNotFound.Error() {
				conf.LoggerLogic().Error("mongo find error : %s", err.Error())
				controllers.ThrowError(c, -1, "首页显示异常，请稍后再试")
			} else {
				output.ID = bson.NewObjectId()
				output.Title = fmt.Sprintf("欢迎您，%s", userInfo.Username)
				output.Username = userInfo.Username
				output.NowTime = time.Now().Format("2006-01-02 15:04:05")

				err := con.Insert(&output)

				if err != nil {
					conf.LoggerLogic().Error("mongo insert error : %s", err.Error())
					controllers.ThrowError(c, -1, "首页显示异常，请稍后再试")
				}
			}
		}

		output.ID = ""
		c.JSON(http.StatusOK, controllers.SwitchResponse(0, output, ""))
	}
}