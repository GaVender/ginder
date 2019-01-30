package main

import (
	"fmt"
)

func main() {
	var _ = fmt.Println
	var ch chan int

	go GetDataFromMongo(2)
	go SendSms(2)
	go UpdateDataToMongo(2)
	go UpdateDataToMongo(2)

	<- ch
}

