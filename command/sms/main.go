package main

import (
	"fmt"
	"ginder/framework/routinepool"
	"time"
)

func main() {
	var _ = fmt.Println

	p, err := routinepool.NewPool(3, 3)

	if err != nil {
		fmt.Println(err.Error())
	}

	for i:=0;i<3;i++ {
		p.Submit(func() error {
			time.Sleep(time.Second * 2)
			fmt.Println(i)
			return nil
		})
	}

	time.Sleep(time.Second * 6)
}
