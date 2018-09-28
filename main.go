package main

import (
	"ginder/routers"
	"ginder/conf"
	"os"
)

func main() {
	startConf()
	routers.Router.Run()
}

func startConf() {
	if "dev" == os.Getenv("env") {
		conf.DevStart()
	} else {
		conf.ProStart()
	}
}
