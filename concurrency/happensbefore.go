package concurrency

import (
	"fmt"
	"time"
)

func main01() {
	c := make(chan int, 10)

	go func() {
		fmt.Println("before sent to channel")
		c <- 1
		fmt.Println("after sent to channel")
	}()

	go func() {
		fmt.Println("before receive from channel")
		d := <-c
		d = -d
		fmt.Println("after received from channel")
	}()

	<-time.After(1 * time.Second)
}
