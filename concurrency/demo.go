package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"
)

// 使用unbuffered chan
func main01() {
	theMine := [5]string{"ore1", "ore2", "ore3"}
	oreChan := make(chan string)

	// Finder
	go func(mine [5]string) {
		for _, m := range mine {
			oreChan <- m
			fmt.Println("finder sent: ", m)
		}
	}(theMine)
	// Ore Breaker
	go func() {
		for i := 0; i < 3; i++ {
			foundOre := <-oreChan
			fmt.Println("Miner: Received " + foundOre + " from finder")
		}
	}()

	<-time.After(time.Second * 5)
}

// 使用buffered chan
func main02() {
	// 创建一个bufferd chan
	bc := make(chan string, 3)
	mines := [5]string{"ore1", "ore2", "ore3"}

	go func(theMines [5]string) {
		for _, mine := range mines {
			if mine == "" {
				break
			}
			bc <- mine
			fmt.Println("finder sent: ", mine)
		}
	}(mines)

	time.Sleep(1 * time.Second)

	go func() {
		for i := 0; i < 3; i++ {
			msg := <-bc
			fmt.Println("Miner: Received " + msg + " from finder")
		}
	}()

	<-time.After(5 * time.Second)
}

// <-doneChan等待阻塞完成
func main03() {
	doneChan := make(chan string)

	go func() {
		doneChan <- "I'm done"
	}()

	// 上面我们等待所有的goroutine执行完使用的是sleep函数
	// 但是这种函数有限，时间不确定，因此，可以使用<-doneChan来等待所有的goroutine执行完
	<-doneChan
}

// 对chan进行非阻塞读
func main04() {
	myChan := make(chan string)

	// 首先创建go routine，因为go routine初始化需要一点时间，因此后面的会继续执行
	go func() {
		myChan <- "Messages!"
	}()

	// select实现对channel的非阻塞读
LABEL1:
	for i := 0; true; i++ {
		select {
		case msg := <-myChan:
			fmt.Println(msg)
			break LABEL1
		default:
			fmt.Println("No message!")
		}
	}

	// select {
	// case msg := <-myChan:
	// 	fmt.Println(msg)
	// default:
	// 	fmt.Println("No messages!")
	// }
}

// select实现非阻塞写
func main05() {
	c := make(chan string)

LABEL1:
	for i := 0; true; i++ {
		select {
		case c <- "message":
			fmt.Println("sent the message")
			break LABEL1
		default:
			fmt.Println("no message sent")
			break LABEL1
		}
	}
}

// go使用互斥锁
func main06() {
	var mutex sync.Mutex
	count := 0

	for i := 0; i < 10; i++ {
		go func() {
			mutex.Lock()
			count++
			fmt.Println(count)
			mutex.Unlock()
		}()
	}
	<-time.After(3 * time.Second)
}

// go使用读写锁
func main07() {
	var rwLock sync.RWMutex
	arr := []int{1, 2, 3}

	go func() {
		fmt.Println("尝试加写锁")
		rwLock.Lock()
		fmt.Println("写锁获得成功")

		arr = append(arr, 4)

		fmt.Println("尝试解除写锁")
		rwLock.Unlock()
		fmt.Println("写锁解除成功")
	}()

	go func() {
		fmt.Println("尝试加读锁")
		rwLock.RLock()
		fmt.Println("读锁获取成功")

		fmt.Println("正在读")
		<-time.After(3 * time.Second)

		fmt.Println("尝试解除读锁")
		rwLock.RUnlock()
		fmt.Println("读锁解除成功")
	}()

	<-time.After(10 * time.Second)
}

// 理解go语言中的context
func main08() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	go handle(ctx, 1500*time.Millisecond)
	select {
	case <-ctx.Done():
		fmt.Println("main", ctx.Err())
	}
}

func handle(ctx context.Context, duration time.Duration) {
	select {
	case <-ctx.Done():
		fmt.Println("handle", ctx.Err())
	case <-time.After(duration):
		fmt.Println("process request within", duration)
	}
}

var status int64

// 理解同步原语中的Cond
func listen(c *sync.Cond) {
	c.L.Lock()
	for atomic.LoadInt64(&status) != 1 {
		c.Wait()
	}
	fmt.Println("listen")
	c.L.Unlock()
}

func broadcast(c *sync.Cond) {
	c.L.Lock()
	atomic.StoreInt64(&status, 1)
	c.Broadcast()
	c.L.Unlock()
}

func main09() {
	c := sync.NewCond(&sync.Mutex{})
	for i := 0; i < 10; i++ {
		go listen(c)
	}
	time.Sleep(1 * time.Second)
	go broadcast(c)

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch
}

// errorgroup处理http请求
func main10() {
	var g errgroup.Group

	urls := []string{
		"https://bilibili.com/",
		"https://baidu.com",
	}

	for _, url := range urls {
		g.Go(func() error {
			resp, err := http.Get(url)
			if err == nil {
				resp.Body.Close()
			}
			return err
		})
	}

	if err := g.Wait(); err == nil {
		fmt.Println("All resquest are fetched!")
	}
}
