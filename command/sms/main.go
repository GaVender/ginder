package main

import (
	"fmt"
	"ginder/framework/routinepool"
	"time"
)

func main() {
	var _ = fmt.Println

	p, err := routinepool.NewPool(10, 3)

	if err != nil {
		fmt.Println(err.Error())
	}

	for i := 0; i < 5; i++ {
		j := i
		p.Submit(func() error {
			time.Sleep(time.Second * 2)
			fmt.Println(j)
			return nil
		})
	}

	fmt.Println(p.RunningAmount())
	time.Sleep(time.Second * 6)
}
