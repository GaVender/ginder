package main

import (
	"fmt"
)

func main() {
	var _ = fmt.Println
	var ch = make(chan int)

	go SendProcedure(2)
	go monitor()

	<-ch
}