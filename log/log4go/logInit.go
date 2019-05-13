package log4go

import (
	log "github.com/jeanphorn/log4go"
)

type Log4Go struct {
	logger log.Logger
}

var logForError Log4Go
var logForLogic Log4Go

func init() {
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
	if err := l.logger.Warn(head + " | " + content); err != nil {
		l.logger.Info("warn", "log warn error: " + err.Error())
	}
}

func (l Log4Go) LogError(head, content string) {
	if err := l.logger.Error(head + " | " + content); err != nil {
		l.logger.Info("error", "log error error: " + err.Error())
	}
}