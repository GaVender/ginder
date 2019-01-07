package main

import (
	"ginder/conf"
	"fmt"
)

func main() {
	var _ = fmt.Println

	l1 := conf.GetLogicLogger()
	defer l1.Close()
	l1.LogInfo("integral", "user is wrong")

	l2 := conf.GetErrorLogger()
	defer l2.Close()
	l2.LogError("controller", "this is wrong")
}
