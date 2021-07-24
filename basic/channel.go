package main

import (
	"fmt"
	"sync"
)

func mmain01() {
	c := make(chan int)
	done := make(chan bool)

	go func() {
		for i := 0; i < 10; i++ {
			fmt.Println(i)
			c <- i
		}
		done <- true
	}()

	go func() {
		for {
			select {
			case <-c:
				continue
			case <-done:
				fmt.Println("done")
				return
			}
		}
	}()

	<-done
	fmt.Println()
}

func main() {
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		for i := 0; i < 10; i++ {
			fmt.Println(i)
		}
		wg.Done()
	}()

	wg.Wait()
}
