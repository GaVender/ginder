package log4go

import (
	"strings"
	"os"

	log "github.com/jeanphorn/log4go"
)

type Log4Go struct {
	logger log.Logger
}

var logForLogic Log4Go
var logForError Log4Go

func init() {
	logError := strings.TrimSpace(os.Getenv("LOG_ERROR"))  // 放置非业务代码的错误的log路径
	logLogic := strings.TrimSpace(os.Getenv("LOG_LOGIC"))  // 放置代码逻辑错误、运行参数、结果等的log路径

	if logError == "" || logLogic == "" {
		panic("日志路径未设置")
	}

	loggerError := log.NewDefaultLogger(log.FINE)
	loggerError.AddFilter("file", log.FINE, log.NewFileLogWriter(logError, true, true))
	logForError = Log4Go{loggerError}

	loggerLogic := log.NewDefaultLogger(log.FINE)
	loggerLogic.AddFilter("file", log.FINE, log.NewFileLogWriter(logLogic, true, true))
	logForLogic = Log4Go{loggerLogic}
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