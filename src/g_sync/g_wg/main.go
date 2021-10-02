package main

import (
	"fmt"
	"sync"
	"time"
)

func ManyWaitForOne() {
	var wg sync.WaitGroup
	wg.Add(1)

	// for i := 0; i < 10; i++ {
	// 	go func (i int)  {
	// 		fmt.Println("routine", i, "正在运行")
	// 		wg.Done()
	// 	}(i)
	// }
	go func ()  {
		fmt.Println("执行了一个任务")
		wg.Done()	
	}()
	// fmt.Println("执行了一个任务")
	// wg.Done()
	<-time.After(1*time.Second)

	for i := 0; i < 2; i++ {
		go func ()  {
			fmt.Println("正在等待")
			wg.Wait()
		}()
	}
	<-time.After(2*time.Second)

	wg.Add(1)
	go func ()  {
		fmt.Println("又执行了一个任务")
		wg.Done()	
	}()
	// fmt.Println("又执行了一个任务")
	// wg.Done()	

	<-time.After(1*time.Second)
}

func main() {
	ManyWaitForOne()
}