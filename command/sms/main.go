package main

import (
	"fmt"

)

func main() {
	var _ = fmt.Println

	GetDataFromMongo(2)
	SendSms(2)
	SendSms(2)
	//UpdateDataToMongo(2)
	//UpdateDataToMongo(2)
}

