package home

import (
	"net/http"
	"fmt"
	"time"

	"github.com/GaVender/ginder/conf"
	"github.com/GaVender/ginder/controllers"

	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
	"gopkg.in/mgo.v2"
)

type homeOutput struct {
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
		output := homeOutput{}

		if err := con.Find(bson.M{"username": userInfo.Username}).One(&output); err != nil {
			if err.Error() != mgo.ErrNotFound.Error() {
				controllers.ThrowError(c, -1, "首页显示异常，请稍后再试")
			} else {
				output.ID = bson.NewObjectId()
				output.Title = fmt.Sprintf("欢迎您，%s", userInfo.Username)
				output.Username = userInfo.Username
				output.NowTime = time.Now().Format("2006-01-02 15:04:05")

				err := con.Insert(&output)

				if err != nil {
					controllers.ThrowError(c, -1, "首页显示异常，请稍后再试")
				}
			}
		}

		output.ID = ""
		c.JSON(http.StatusOK, controllers.SwitchResponse(0, output, ""))
	}
}