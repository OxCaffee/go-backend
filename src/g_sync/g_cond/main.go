package main

import (
	"fmt"
	"sync"
)

func SingleCondWaitAndSignal() {
	var mux sync.Mutex
	cond := sync.NewCond(&mux)

	n := 3
	running := make(chan bool, n)
	awaking := make(chan bool, n)

	for i := 0; i < n; i++ {
		go func ()  {
			// mux加锁
			mux.Lock()
			// 发送正在running
			running <- true
			fmt.Println("当前go协程正在运行")
			// 让出当前cpu占用权同时解锁
			cond.Wait()
			// 返回运行
			awaking <- true
			fmt.Println("当前go协程已经被唤醒")
			// 解锁
			mux.Unlock()
		}()
	}

	// 将所有的running通道信息接收
	for i := 0; i < n; i++ {
		<-running
	}

	// 依次唤醒
	for i := 0; i < n; i++ {
		select {
		case <-awaking:
			// 此时说明协程并没有陷入休眠
			fmt.Println("协程并未休眠")
		default:
		}

		mux.Lock()
		cond.Signal()
		mux.Unlock()

		<-awaking
		select {
		case <-awaking:
			fmt.Println("太多需要唤醒的go协程了")
		default:
		}
	}
	cond.Signal()
}

func main() {
	SingleCondWaitAndSignal()
}
