package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type MyOnce struct {
	mux sync.Mutex
	done uint32
}

// func (o *MyOnce) Do(f func()) {
// 	// 这里实现和sync原生包不太一样，官方说这里存在问题，但是测试并没有发现
// 	if atomic.CompareAndSwapUint32(&o.done, 0, 1) {
// 		f()
// 	}
// }

func (o *MyOnce) Do(f func()) {
	if atomic.LoadUint32(&o.done) == 0 {
		o.doSlow(f)
	}
}

func (o *MyOnce) doSlow(f func()) {
	o.mux.Lock()
	defer o.mux.Unlock()

	if o.done == 0 {
		f()
		defer atomic.StoreUint32(&o.done, 1)
	}
}

func main() {
	once := MyOnce{mux: sync.Mutex{}, done: 0}

	var f = func() {
		fmt.Println("executed one time")
	}

	for i := 0; i < 10000; i++ {
		go func ()  {
			once.Do(f)	
		}()
	}

	<-time.After(5*time.Second)
}