package main

import (
	"fmt"
	"ginder/framework/routinepool"
)

func main() {
	var _ = fmt.Println

	p, err := routinepool.NewPool(2, 3)

	if err != nil {
		fmt.Println(err.Error())
	}

	p.Submit(func() error {
		fmt.Println("1")
		return nil
	})
}
