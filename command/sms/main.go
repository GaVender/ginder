package main

import (
	"fmt"
)

func main() {
	var _ = fmt.Println
	var ch chan int
	var platform uint8 = 2

	go GetDataFromMongo(platform)
	go CreateSendPool(platform)
	go CreateUpdatePool(platform)

	<- ch
}

