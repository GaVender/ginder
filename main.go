package main

import (
	_ "ginder/conf"
	//_ "ginder/routers"
	"ginder/command"
)

func main() {
	command.AutoClean()
}


