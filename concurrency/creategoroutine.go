package main

import (
	"fmt"
	"time"
)

func countNumber1() {
	for i := 1; i <= 10; i++ {
		fmt.Println(i)
		time.Sleep(500 * time.Millisecond)
	}
}

func countNumber2() {
	for i := 101; i <= 110; i++ {
		fmt.Println(i)
		time.Sleep(600 * time.Millisecond)
	}
}

func main() {
	go countNumber1()
	go countNumber2()
	fmt.Println("over!")
}
