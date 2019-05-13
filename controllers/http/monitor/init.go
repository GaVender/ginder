package monitor

import "github.com/xiaobai22/gokit-service/monitorkit"

const (
	Port = "9091"
	Url  = "http://127.0.0.1:" + Port + "/black"
)

func init() {
	go monitorkit.StartMonitorBB(Port, "/black")
}