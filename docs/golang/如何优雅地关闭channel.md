# 如何优雅地关闭channel

## 关闭channel的难点

* 在不改变channel自身状态下，无法获知一个channel是否关闭
* 关闭一个closed channel会导致panic。所以，如果关闭channel的一方在不知道channel是否关闭的情况下就去贸然关闭channel是很危险的事情
* 向一个closed channel发送数据会导致panic。所以，如果向channel发送数据的一方不知道channel被关闭就去发送数据很危险

## 关闭channel的原则

**不要从一个 receiver 侧关闭 channel，也不要在有多个 sender 时，关闭 channel。**

## 不优雅的关闭方法

### defer-recover兜底

使用defer-recover机制，放心大胆关闭channel或者向channel发送数据。即使发生了panic，有defer-recover机制兜底

### 使用sync.Once保证只关闭一次

利用sync.Once特性保证只关闭channel一次。**这种方法保证了channel只被关闭一次，但是无法保证被关闭后没有sender发送数据**

## 优雅的关闭方法

**增加一个关闭信号的channel，receiver通过信号channel下达关闭数据channel指令，sender接收到关闭信号之后，停止接收数据。**

### 一个sender，一个receiver

直接从sender端关闭即可

### 一个sender，M个receiver

直接从sender端关闭即可

### N个sender，一个receiver

在这种情况下，优雅地关闭channel的方法就是receiver负责通知sender不要再发送数据了，然后关闭它。

```go
func main() {
    datach := make(chan int, 100)
    stopch := make(chan struct{})
    
    // senders
    for i := 0; i < 100; i++ {
        go func() {
            for {
                select {
                case <-stopch:
                	return
                case datach <-rand.Intn(100):
                }
            }  
        }()
    }
    
    // receiver
    go func() {
        for val := range datach {
            if val == 100 {
                fmt.Println("please stop the sender")
                // 关闭stopch，让gc代劳
                close(stopch)
                return
            }
            fmt.Println(val)
        }  
    }()
    
    <-time.After(time.Second)
}
```

### N个sender，M个receiver

这里有 M 个 receiver，如果直接还是采取第 3 种解决方案，由 receiver 直接关闭 stopCh 的话，就会重复关闭一个 channel，导致 panic。因此需要增加一个中间人，M 个 receiver 都向它发送关闭 dataCh 的“请求”，中间人收到第一个请求后，就会直接下达关闭dataCh 的指令（通过关闭 stopCh，这时就不会发生重复关闭的情况，因为 stopCh 的发送方只有 中间人一个）。另外，这里的 N 个 sender 也可以向中间人发送关闭 dataCh 的请求。

```go
func CloseChanElegantly() {
	// datach := make(chan int, 100)
	stopch := make(chan struct{})
	tostop := make(chan string, 1)

	var stopsignal string

	go func() {
		stopsignal = <-tostop
		fmt.Println(stopsignal)
		close(stopch)
	}()

	for i := 0; i < 3; i++ {
		go func(i int) {
			fmt.Printf("这是第%d个sender\n", i)
			if i == 2 {
				fmt.Println("第2个sender决定关闭channel")
				tostop <- "stop!!!!!"
			}

			select {
			case <-tostop:
				return
			default:
			}
		}(i)
	}

	for i := 0; i < 4; i++ {
		go func(i int) {
			fmt.Printf("这是第%d个receiver\n", i)
			if i == 2 {
				fmt.Println("第2个receiver决定关闭channel")
				tostop <- "stop!!!!!"
			}
			select {
			case <-tostop:
				return
			default:
			}
		}(i)
	}

	<-time.After(3*time.Second)
}
```

