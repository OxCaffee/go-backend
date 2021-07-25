package main

import "fmt"

func nilmain() {
	var a = []int(nil)
	var b = map[string]int(nil)
	var c = (*[5]int)(nil)
	var d = chan int(nil)

	for k, v := range a {
		fmt.Println("key:", k, "; value:", v)
	}

	for k, v := range b {
		fmt.Println("key:", k, "; value:", v)
	}

	// // 这个会触发panic
	for k, v := range c {
		fmt.Println("key:", k, "; value:", v)
	}

	// 如果不适用value，则不会panic
	for k, _ := range c {
		fmt.Println("key:", k)
	}

	// deadlock，永久阻塞
	for range d {
		fmt.Println("chan!!!")
	}
}
