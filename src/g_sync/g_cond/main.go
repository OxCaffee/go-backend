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
	// cond.Signal()
}

func CondSignalGeneration() {
	mux := sync.Mutex{}
	cd := sync.NewCond(&mux)
	running := make(chan bool, 100)
	awake := make(chan int, 100)

	for i := 0; i < 100; i++ {
		go func (i int)  {
			mux.Lock()
			running <-true
			fmt.Println("协程", i, "陷入休眠")
			cd.Wait()
			fmt.Println("协程", i, "被唤醒")
			awake <- i
			mux.Unlock()
		}(i)

		if i > 0 {
			prev := <-awake
			if prev != i - 1 {
				fmt.Println("唤醒了错误的go协程")
			}
		}
		<-running
		mux.Lock()
		cd.Signal()
		mux.Unlock()
	}
}

func CondBroadcast() {
	mux := sync.Mutex{}
	cond := sync.NewCond(&mux)
	n := 200
	running := make(chan int, n)
	awake := make(chan int, n)
	exit := false

	for i := 0; i < n; i++ {
		go func (i int)  {
			mux.Lock()
			for !exit {
				running <- i
				cond.Wait()
				awake <- i
			}
			mux.Unlock()
		}(i)
	}

	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			<-running
		}

		if i == n - 1 {
			mux.Lock()
			exit = true
			mux.Unlock()
		}

		select {
		case <-awake:
			fmt.Println("协程并未休眠")
		default:
		}

		mux.Lock()
		cond.Broadcast()
		mux.Unlock()

		seen := make([]bool, n)
		for i := 0; i < n; i++ {
			g := <-awake
			if seen[g] {
				fmt.Println("同一个协程唤醒两次:", g)
			}
			seen[g] = true
		}
	}

	select {
	case <-running:
		fmt.Println("协程并未退出")
	default:
	}
}

func main() {
	// SingleCondWaitAndSignal()
	// CondSignalGeneration()
	CondBroadcast()
}
