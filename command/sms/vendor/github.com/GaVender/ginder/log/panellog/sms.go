package panellog

import "github.com/xiaobai22/gokit-service/blackboardkit"

var SmsPanelLog *blackboardkit.BlackBoradKit

func init() {
	SmsPanelLog = blackboardkit.NewBlockBorad()

	SmsPanelLog.InitLogKit("panic", "monitor", "getSms", "getSmsError", "sendSms", "sendSmsError", "updateSms",
		"updateSmsError")
	SmsPanelLog.SetLogReadme("panic", "panic日志")
	SmsPanelLog.SetLogReadme("monitor", "程序监控日志")
	SmsPanelLog.SetLogReadme("getSms", "获取短信")
	SmsPanelLog.SetLogReadme("getSmsError", "获取短信出错")
	SmsPanelLog.SetLogReadme("sendSms", "发送短信")
	SmsPanelLog.SetLogReadme("sendSmsError", "发送短信出错")
	SmsPanelLog.SetLogReadme("updateSms", "更改短信")
	SmsPanelLog.SetLogReadme("updateSmsError", "更改短信出错")

	SmsPanelLog.SetNoPrintToConsole(true)
	SmsPanelLog.Ready()
}
