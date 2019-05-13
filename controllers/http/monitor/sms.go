package monitor

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Sms(c *gin.Context) {
	w := c.Writer
	result, err := http.Get(Url)

	if err != nil {
		_, _ = w.WriteString(fmt.Sprintf("面板日志获取出错：%s request error: %s", Url, err.Error()))
		return
	}

	defer func() {
		if err := result.Body.Close(); err != nil {
			_, _ = w.WriteString(fmt.Sprintf("面板日志获取出错：%s close error: %s", Url, err.Error()))
		}
	}()

	content2, err := ioutil.ReadAll(result.Body)

	if err != nil {
		_, _ = w.WriteString("面板日志获取出错：" + err.Error())
	}

	_, _ = w.WriteString("面板日志获取出错：" + string(content2))
}