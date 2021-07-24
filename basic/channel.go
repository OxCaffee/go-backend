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

func mmain02() {
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

func mmain03() {
	ch1 := make(chan int, 1)
	ch2 := make(chan int, 1)

	ch1 <- 3
	ch2 <- 5

	select {
	case <-ch1:
		fmt.Println("ch1 selected.")
		break
		fmt.Println("ch1 selected after break")
	case <-ch2:
		fmt.Println("ch2 selected.")
		fmt.Println("ch2 selected without break")
	}
}

func main() {
	c := make(chan int, 10)

	close(c)

	c <- 1
}
