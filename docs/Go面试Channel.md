# Go面试——Channel

<!-- vscode-markdown-toc -->
* 1. [channel的合理退出](#channel)
	* 1.1. [select语句](#select)
	* 1.2. [WaitGroup](#WaitGroup)
* 2. [如何结束select](#select-1)
* 3. [能否从已关闭的channel中读取数据](#channel-1)
* 4. [能否向已经关闭的channel写数据](#channel-1)
* 5. [能否对未初始化的channel读写](#channel-1)
	* 5.1. [答案](#)
	* 5.2. [demo](#demo)
	* 5.3. [解析](#-1)

<!-- vscode-markdown-toc-config
	numbering=true
	autoSave=true
	/vscode-markdown-toc-config -->
<!-- /vscode-markdown-toc -->

##  1. <a name='channel'></a>channel的合理退出

###  1.1. <a name='select'></a>select语句

```go
func main() {
	c := make(chan int)
	done := make(chan bool)

    // go1
	go func() {
		for i := 0; i < 10; i++ {
			fmt.Println(i)
			c <- i
		}
		done <- true
	}()

    // go2
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
```

上面的代码片段开辟了两个goroutine，我们分别称为go1和go2，go1负责生产10个数，并且推送到[`channel c`]()中，go2负责[`channel c`]()中取出数字，并在接收到[`done`]()信号时退出，看似没有什么问题，运行一下，结果如下：

```go
PS D:\Github\repo\Go-Backend> go run "d:\Github\repo\Go-Backend\basic\channel.go"
0
1
2
3
4
5
6
7
8
9
```

可以看到[`"done"`]()并没有被打印，为什么呢？

**当go1的最后一个数字9被推送到[`channel c`]() 中的时候，[`true`]()信号会被紧接着推送到[`channel done`]()中，此时main中的[`<-done`]()已经阻塞了很久，接收到信号之后立马恢复运行，程序退出，而此时go2才刚刚阻塞，主协程main挂掉，剩下的子协程接着挂掉** 。

###  1.2. <a name='WaitGroup'></a>WaitGroup

```go
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
```

##  2. <a name='select-1'></a>如何结束select

使用[`break`]()

```go
func main() {
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
```

##  3. <a name='channel-1'></a>能否从已关闭的channel中读取数据

```go
func main() {
	c := make(chan int)

	go func() {
		c <- 1
		close(c)
	}()

	<-time.After(time.Second)

	for {
		go func() {
			data, ok := <-c
			fmt.Println("data:", data, "; ok:", ok)
		}()

		<-time.After(time.Second)
	}

	<-time.After(time.Second)
}
```

输出结果：

```go
data: 1 ; ok: true
data: 0 ; ok: false
data: 0 ; ok: false
data: 0 ; ok: false
data: 0 ; ok: false
data: 0 ; ok: false
data: 0 ; ok: false
data: 0 ; ok: false
data: 0 ; ok: false
```

可以看到，读**已经关闭**的 `chan` 能一直读到东西，但是读到的内容根据通道内`关闭前`是否有元素而不同。

- 如果 `chan` 关闭前，`buffer` 内有元素**还未读** , 会正确读到 `chan` 内的值，且返回的第二个 bool 值（是否读成功）为 `true`。
- 如果 `chan` 关闭前，`buffer` 内有元素**已经被读完**，`chan` 内无值，接下来所有接收的值都会非阻塞直接成功，返回 `channel` 元素的**零值**，但是第二个 `bool` 值一直为 `false`。

##  4. <a name='channel-1'></a>能否向已经关闭的channel写数据

```go
func main() {
	c := make(chan int, 10)

	close(c)

	c <- 1
}
```

运行结果：

```go
panic: send on closed channel
```

可以看到，当写入已经关闭的channel时，会触发panic

##  5. <a name='channel-1'></a>能否对未初始化的channel读写

###  5.1. <a name=''></a>答案

读写未初始化的[`chan`]()会阻塞。

###  5.2. <a name='demo'></a>demo

* **写未初始化的 chan**

  ```go
  package main
  // 写未初始化的chan
  func main() {
   var c chan int
   c <- 1
  }
  ```

  ```go
  // 输出结果
  fatal error: all goroutines are asleep - deadlock!
  
  goroutine 1 [chan send (nil chan)]:
  main.main()
          /Users/admin18/go/src/code.byted.org/linzhaolun/repos/main.go:6 +0x36
  ```

* **读未初始化的channel**

  ```go
  package main
  import "fmt"
  // 读未初始化的chan
  func main() {
   var c chan int
   num, ok := <-c
   fmt.Printf("读chan的协程结束, num=%v, ok=%v\n", num, ok)
  }
  ```

  ```go
  // 输出结果
  fatal error: all goroutines are asleep - deadlock!
  
  goroutine 1 [chan receive (nil chan)]:
  main.main()
          /Users/admin18/go/src/code.byted.org/linzhaolun/repos/main.go:6 +0x46
  ```

###  5.3. <a name='-1'></a>解析

* **为什么未初始化的channel写会阻塞**

```go
//在 src/runtime/chan.go中
func chansend(c *hchan, ep unsafe.Pointer, block bool, callerpc uintptr) bool {
 if c == nil {
      // 不能阻塞，直接返回 false，表示未发送成功
      if !block {
        return false
      }
      gopark(nil, nil, waitReasonChanSendNilChan, traceEvGoStop, 2)
      throw("unreachable")
 }
  // 省略其他逻辑
}
```

- 未初始化的 `chan` 此时是等于 `nil`，当它不能阻塞的情况下，直接返回 `false`，表示写 `chan` 失败
- 当 `chan` 能阻塞的情况下，则直接阻塞 `gopark(nil, nil, waitReasonChanSendNilChan, traceEvGoStop, 2)`, 然后调用 `throw(s string)` 抛出错误，其中 `waitReasonChanSendNilChan` 就是刚刚提到的报错 `"chan send (nil chan)"`



* **为什么读未初始化的channel会阻塞**

```go
//在 src/runtime/chan.go中
func chanrecv(c *hchan, ep unsafe.Pointer, block bool) (selected, received bool) {
    //省略逻辑...
    if c == nil {
        if !block {
          return
        }
        gopark(nil, nil, waitReasonChanReceiveNilChan, traceEvGoStop, 2)
        throw("unreachable")
    }
    //省略逻辑...
} 
```

- 未初始化的 `chan` 此时是等于 `nil`，当它不能阻塞的情况下，直接返回 `false`，表示读 `chan` 失败
- 当 `chan` 能阻塞的情况下，则直接阻塞 `gopark(nil, nil, waitReasonChanReceiveNilChan, traceEvGoStop, 2)`, 然后调用 `throw(s string)` 抛出错误，其中 `waitReasonChanReceiveNilChan` 就是刚刚提到的报错 `"chan receive (nil chan)"`
