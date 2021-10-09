package main

import "fmt"

func CodeAndThread() {
	for i := 0; i < 10000; i++ {
		go func() {
			for {
				fmt.Print(".")
			}
		}()
	}

	ch := make(chan int, 1)
	go func() {
		fmt.Println("hello")
		ch <- 1
	}()

	<-ch
}

func main() {
	CodeAndThread()
}
