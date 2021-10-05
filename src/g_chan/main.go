package main

import (
	"fmt"
	"time"
)

func CloseChan() {
	ch := make(chan int)

	go func ()  {
		<-ch
		fmt.Println("不被阻塞了")
	}()

	close(ch)
	// fmt.Println(<-ch)
	// fmt.Println(ch)
	<-time.After(time.Second)
}

func CloseChanElegantly() {
	// datach := make(chan int, 100)
	stopch := make(chan struct{})
	tostop := make(chan string, 1)

	var stopsignal string

	go func() {
		stopsignal = <-tostop
		fmt.Println(stopsignal)
		close(stopch)
	}()

	for i := 0; i < 3; i++ {
		go func(i int) {
			fmt.Printf("这是第%d个sender\n", i)
			if i == 2 {
				fmt.Println("第2个sender决定关闭channel")
				tostop <- "stop!!!!!"
			}

			select {
			case <-tostop:
				return
			default:
			}
		}(i)
	}

	for i := 0; i < 4; i++ {
		go func(i int) {
			fmt.Printf("这是第%d个receiver\n", i)
			if i == 2 {
				fmt.Println("第2个receiver决定关闭channel")
				tostop <- "stop!!!!!"
			}
			select {
			case <-tostop:
				return
			default:
			}
		}(i)
	}

	<-time.After(3*time.Second)
}

func main() {
	CloseChanElegantly()
}