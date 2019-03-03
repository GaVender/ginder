package monitor

import (
	"fmt"
	"net/http"
	"io/ioutil"

	"gitlab.etcchebao.cn/go_service/api.sms/conf"

	"github.com/gin-gonic/gin"
)

func Sms(c *gin.Context) {
	w := c.Writer

	fmt.Fprintln(w, "---------------- 配置监控：---------------- \n")

	if true == conf.ErrorLogStart {
		fmt.Fprintln(w, "系统错误日志启动成功......")
	} else {
		fmt.Fprintln(w, "系统错误日志启动失败......")
	}

	if true == conf.LogicLogStart {
		fmt.Fprintln(w, "逻辑错误日志启动成功......")
	} else {
		fmt.Fprintln(w, "逻辑错误日志启动失败......")
	}

	if true == conf.MysqlMasterStart {
		fmt.Fprintln(w, "mysql 主库启动成功......")
	} else {
		fmt.Fprintln(w, "mysql 从库启动失败......")
	}

	if true == conf.MysqlSlaveStart {
		fmt.Fprintln(w, "mysql 从库启动成功......")
	} else {
		fmt.Fprintln(w, "mysql 从库启动失败......")
	}

	if true == conf.RedisMasterStart {
		fmt.Fprintln(w, "redis 主库启动成功......")
	} else {
		fmt.Fprintln(w, "redis 主库启动失败......")
	}

	if true == conf.RedisSlaveStart {
		fmt.Fprintln(w, "redis 从库启动成功......")
	} else {
		fmt.Fprintln(w, "redis 从库启动失败......")
	}

	if true == conf.MongoStart {
		fmt.Fprintln(w, "mongo 启动成功......")
	} else {
		fmt.Fprintln(w, "mongo 启动失败......")
	}

	fmt.Fprintln(w)

	result, err := http.Get("http://127.0.0.1:9091/black")
	defer result.Body.Close()
	content2, err := ioutil.ReadAll(result.Body)

	if err != nil {
		fmt.Fprintln(w, "面板日志获取出错：", err)
	}

	fmt.Fprintln(w, string(content2))
}