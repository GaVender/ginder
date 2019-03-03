package log4go

import (
	"strings"
	"os"

	log "github.com/jeanphorn/log4go"
)

type Log4Go struct {
	logger log.Logger
}

var logForError Log4Go
var logForLogic Log4Go

func init() {
	// 创建目录或文件的没改
	logError := strings.TrimSpace(os.Getenv("LOG_ERROR"))  // 放置非业务代码的错误的log路径
	logLogic := strings.TrimSpace(os.Getenv("LOG_LOGIC"))  // 放置代码逻辑错误、运行参数、结果等的log路径

	if logError == "" || logLogic == "" {
		panic("日志路径未设置")
	}

	errorLog := log.NewDefaultLogger(log.FINE)
	errorLog.LoadJsonConfiguration("./log/log4go/error_config.json")
	logForError = Log4Go{errorLog}

	logicLog := log.NewDefaultLogger(log.FINE)
	logicLog.LoadJsonConfiguration("./log/log4go/logic_config.json")
	logForLogic = Log4Go{logicLog}
}

func GetErrorLogger() Log4Go {
	return logForError
}

func GetLogicLogger() Log4Go {
	return logForLogic
}

func (l Log4Go) Close() {
	l.logger.Close()
}

func (l Log4Go) LogInfo(head, content string) {
	l.logger.Info(head + " | " + content)
}

func (l Log4Go) LogWarn(head, content string) {
	l.logger.Warn(head + " | " + content)
}

func (l Log4Go) LogError(head, content string) {
	l.logger.Error(head + " | " + content)
}