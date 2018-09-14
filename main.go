package main

import (
	log "github.com/jeanphorn/log4go"
)

func main() {
	log.LoadConfiguration("./log/log_config.json")

	log.LOGGER("Test").Info("category Test info test message: %s", "new test msg")

	log.Close()
}
