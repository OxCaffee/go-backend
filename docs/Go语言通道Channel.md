# Go源码阅读——runtime.chan

<!-- vscode-markdown-toc -->
* 1. [chan结构体定义](#chan)
* 2. [chan的创建](#chan-1)
	* 2.1. [#真正创建chan——#makechan](#chanmakechan)
* 3. [发送数据](#)
	* 3.1. [发送数据前检查——#chansend](#chansend)
	* 3.2. [如果有goroutine急着接收数据——#send](#goroutinesend)
	* 3.3. [发送数据总结](#-1)
* 4. [接收数据](#-1)
	* 4.1. [如果有goroutine急着发送数据——#recv](#goroutinerecv)
	* 4.2. [接收数据总结](#-1)
* 5. [关闭chan](#chan-1)
	* 5.1. [关闭chan总结](#chan-1)

<!-- vscode-markdown-toc-config
	numbering=true
	autoSave=true
	/vscode-markdown-toc-config -->
<!-- /vscode-markdown-toc -->

##  1. <a name='chan'></a>chan结构体定义

```go
type hchan struct {
	qcount   uint           // 队列中元素总数量
	dataqsiz uint           // 循环队列的长度
	buf      unsafe.Pointer // 指向长度为 dataqsiz 的底层数组，只有在有缓冲时这个才有意义
	elemsize uint16         // 能够发送和接受的元素大小
	closed   uint32         // 是否关闭
	elemtype *_type // 元素的类型
	sendx    uint   // 当前已发送的元素在队列当中的索引位置
	recvx    uint   // 当前已接收的元素在队列当中的索引位置
	recvq    waitq  // 接收 Goroutine 链表
	sendq    waitq  // 发送 Goroutine 链表

	lock mutex // 互斥锁
}

// waitq 是一个双向链表，里面保存了 goroutine
type waitq struct {
	first *sudog
	last  *sudog
}
```

<div align=center><img src="/assets/chan1.png"/></div>

可以看到,`chan`底层就是一个循环队列。

##  2. <a name='chan-1'></a>chan的创建

在 Go 中我们使用 `make(chan T, cap)` 来创建 channel，make 语法会在编译时，转换为 `makechan64` 和 `makechan`：

```go
func makechan64(t *chantype, size int64) *hchan {
	if int64(int(size)) != size {
		panic(plainError("makechan: size out of range"))
	}

	return makechan(t, int(size))
}
```

`makechan64` 主要是做了一下检查，最终还是会调用 `makechan` ，在看 `makechan` 源码之前，我们先来看两个全局常量，接下来会用到：

```go
const (
	maxAlign  = 8
	hchanSize = unsafe.Sizeof(hchan{}) + uintptr(-int(unsafe.Sizeof(hchan{}))&(maxAlign-1))
)
```

- `maxAlign` 是内存对齐的最大值，这个等于 64 位 CPU 下的 cacheline 的大小
- `hchanSize` 计算 `unsafe.Sizeof(hchan{})` 最近的 8 的倍数

###  2.1. <a name='chanmakechan'></a>#真正创建chan——#makechan

```go
func makechan(t *chantype, size int) *hchan {
	elem := t.elem

	// 先做一些检查
    // 元素大小不能大于等于 64k
	if elem.size >= 1<<16 {
		throw("makechan: invalid channel element type")
	}
    // 判断当前的 hchanSize 是否是 maxAlign 整数倍，并且元素的对齐大小不能大于最大对齐的大小
	if hchanSize%maxAlign != 0 || elem.align > maxAlign {
		throw("makechan: bad alignment")
	}

    // 这里计算内存是否超过限制
	mem, overflow := math.MulUintptr(elem.size, uintptr(size))
	if overflow || mem > maxAlloc-hchanSize || size < 0 {
		panic(plainError("makechan: size out of range"))
	}

	var c *hchan
	switch {
	case mem == 0: // 如果是无缓冲通道
		c = (*hchan)(mallocgc(hchanSize, nil, true)) // 为 hchan 分配内存
		c.buf = c.raceaddr() // 这个是 for data race 检测的
	case elem.ptrdata == 0: // 元素不包含指针
		c = (*hchan)(mallocgc(hchanSize+mem, nil, true)) // 为 hchan 和底层数组分配一段连续的内存地址
		c.buf = add(unsafe.Pointer(c), hchanSize)
	default: // 如果元素包含指针，分别为 hchan 和 底层数组分配内存地址
		c = new(hchan)
		c.buf = mallocgc(mem, elem, true)
	}

    // 初始化一些值
	c.elemsize = uint16(elem.size)
	c.elemtype = elem
	c.dataqsiz = uint(size)
	lockInit(&c.lock, lockRankHchan)

	return c
}
```

因此，创建的过程可以总结为如下：

- 创建时会做一些检查
  - 元素大小不能超过 64K
  - 元素的对齐大小不能超过 maxAlign 也就是 8 字节
  - 计算出来的内存是否超过限制
- 创建时的策略
  - 如果是无缓冲的 channel，会直接给 hchan 分配内存
  - 如果是有缓冲的 channel，并且元素不包含指针，那么会为 hchan 和底层数组分配一段连续的地址
  - 如果是有缓冲的 channel，并且元素包含指针，那么会为 hchan 和底层数组分别分配地址

##  3. <a name=''></a>发送数据

<div align=center><img src="/assets/chan2.png"/></div>

###  3.1. <a name='chansend'></a>发送数据前检查——#chansend

我们在 `x <- chan T` 进行发送数据的时候最终会被编译成 `chansend1`：

```go
func chansend1(c *hchan, elem unsafe.Pointer) {
	chansend(c, elem, true, getcallerpc())
}
```

而 `chansend1` 最终还是调用了 `chansend` 主要的逻辑都在 `chansend` 上面，注意看下方源码和注释：

```go
// 代码中删除了调试相关的代码
func chansend(c *hchan, ep unsafe.Pointer, block bool, callerpc uintptr) bool {
    // 如果是一个 nil 值的 channel
    // 如果是非阻塞的话就直接返回
    // 如果不是，那么则调用 gopark 休眠当前 goroutine 并且抛出 panic 错误
	if c == nil {
		if !block {
			return false
		}
		gopark(nil, nil, waitReasonChanSendNilChan, traceEvGoStop, 2)
		throw("unreachable")
	}

    // fast path 如果当前是非阻塞的
    // 并且通道尚未关闭
    // 并且缓冲区已满时，直接返回
	if !block && c.closed == 0 && full(c) {
		return false
	}

    // 加锁
	lock(&c.lock)

    // 如果通道已经关闭了，直接 panic，不允许向一个已经关闭的 channel 写入数据
	if c.closed != 0 {
		unlock(&c.lock)
		panic(plainError("send on closed channel"))
	}

    // 如果当前存在等待接收数据的 goroutine 直接取出第一个，将数据传递给第一个等待的 goroutine
	if sg := c.recvq.dequeue(); sg != nil {
		// send 用于发送数据，我们后面再看
		send(c, sg, ep, func() { unlock(&c.lock) }, 3)
		return true
	}

	// 如果当前 channel 包含缓冲区，并且缓冲区没有满
	if c.qcount < c.dataqsiz {
		// 计算数组中下一个可以存放数据的地址
		qp := chanbuf(c, c.sendx)

        // 将当前的数据放到缓冲区中
		typedmemmove(c.elemtype, qp, ep)

        // 索引加一
        c.sendx++

        // 由于是循环队列，如果索引地址等于数组长度，就需要将索引移动到 0
		if c.sendx == c.dataqsiz {
			c.sendx = 0
		}

        // 当前缓存数据量加一
		c.qcount++
		unlock(&c.lock)
		return true
	}

    // 如果是非阻塞的就直接返回了，因为非阻塞发送的情况已经走完了，下面是阻塞发送的逻辑
	if !block {
		unlock(&c.lock)
		return false
	}

	// 获取发送数据的 goroutine
	gp := getg()
    // 获取 sudog 结构体，并且设置相关信息，包括当前的 channel，是否是 select 等
	mysg := acquireSudog()
	mysg.releasetime = 0
	if t0 != 0 {
		mysg.releasetime = -1
	}
	mysg.elem = ep
	mysg.waitlink = nil
	mysg.g = gp
	mysg.isSelect = false
	mysg.c = c
	gp.waiting = mysg
	gp.param = nil

    // 将 sudog 结构加入到发送的队列中
	c.sendq.enqueue(mysg)

    // 挂起当前 goroutine 等待接收 channel数据
	gopark(chanparkcommit, unsafe.Pointer(&c.lock), waitReasonChanSend, traceEvGoBlockSend, 2)

    // 保证当前数据处于活跃状态避免被回收
	KeepAlive(ep)

	// 发送者 goroutine 被唤醒，检查当前 sg 的状态
	if mysg != gp.waiting {
		throw("G waiting list is corrupted")
	}
	gp.waiting = nil
	gp.activeStackChans = false
	if gp.param == nil {
		if c.closed == 0 {
			throw("chansend: spurious wakeup")
		}
		panic(plainError("send on closed channel"))
	}
	gp.param = nil
	if mysg.releasetime > 0 {
		blockevent(mysg.releasetime-t0, 2)
	}

    // 取消 channel 绑定
	mysg.c = nil
    // 释放 sudog
	releaseSudog(mysg)
	return true
}
```

###  3.2. <a name='goroutinesend'></a>如果有goroutine急着接收数据——#send

```go
func send(c *hchan, sg *sudog, ep unsafe.Pointer, unlockf func(), skip int) {
    // 如果 sudog 上存在数据元素，就调用 sendDirect 直接把数据拷贝到接收变量的地址上
	if sg.elem != nil {
		sendDirect(c.elemtype, sg, ep)
		sg.elem = nil
	}
	gp := sg.g
	unlockf()
	gp.param = unsafe.Pointer(sg)
	if sg.releasetime != 0 {
		sg.releasetime = cputicks()
	}

    // 调用 goready 将接受者的 Goroutine 标记为可运行状态，并把它放到发送方的所在处理器的 runnext 等待执行，下次调度时就会执行到它。
    // 注意这里不是立即执行
	goready(gp, skip+1)
}
```

###  3.3. <a name='-1'></a>发送数据总结

向 channel 中发送数据时大概分为两大块，检查和数据发送，而数据发送又分为三种情况：

- 如果 channel 的`recvq` 存在阻塞等待的接收数据的 goroutine 那么将会直接将数据发送给第一个等待的 goroutine
  - 这里会直接将数据拷贝到 `x <-ch` 接收者的变量 `x` 上
  - 然后将接收者的 Goroutine 修改为可运行状态，并把它放到发送方所在处理器的 runnext 上等待下一次调度时执行。
- 如果 channel 是有缓冲的，并且缓冲区没有满，这个时候就会把数据放到缓冲区中
- 如果 channel 的缓冲区满了，这个时候就会走阻塞发送的流程，获取到 sudog 之后将当前 Goroutine 挂起等待唤醒，唤醒后将相关的数据解绑，回收掉 sudog

##  4. <a name='-1'></a>接收数据

<div align=center><img src="/assets/chan3.png"/></div>

在 Go 中接收 channel 数据有两种方式

- `x <- ch` 编译时会被转换为 `chanrecv1`
- `x, ok <- ch` 编译时会被转换为 `chanrecv2`

`chanrecv1` 和 `chanrecv2` 没有多大区别，只是 `chanrecv2` 比 `chanrecv1` 多了一个返回值，最终都是调用的 `chanrecv` 来实现的接收数据

```go
// selected 用于 select{} 语法中是否会选中该分支
// received 表示当前是否真正的接收到数据，用来判断 channel 是否 closed 掉了
func chanrecv(c *hchan, ep unsafe.Pointer, block bool) (selected, received bool) {
	// 和发送数据类似，先判断是否为nil，如果是 nil 并且阻塞接收就会 panic
	if c == nil {
        // 如果不阻塞，直接返回
		if !block {
			return
		}
        // 如果是阻塞模式，接收一个nil的chan会导致挂起
		gopark(nil, nil, waitReasonChanReceiveNilChan, traceEvGoStop, 2)
        // 不会执行到这里
		throw("unreachable")
	}

	// Fast path: 检查非阻塞的操作
    // empty 主要是有两种情况返回 true:
    // 1. 无缓冲channel，并且没有阻塞住发送者
    // 2. 有缓冲 channel，但是缓冲区没有数据
	if !block && empty(c) {
		// 这里判断通道是否关闭，如果是未关闭的通道说明当前还没准备好数据，直接返回
		if atomic.Load(&c.closed) == 0 {
			return
		}
		// 如果通道已经关闭了，再检查一下通道还有没有数据，如果已经没数据了，我们清理到 ep 指针中的数据并且返回
		if empty(c) {
			if ep != nil {
				typedmemclr(c.elemtype, ep)
			}
			return true, false
		}
	}

	// 上锁
	lock(&c.lock)

    // 如果通道已经关闭了，并且已经没数据了，我们清理到 ep 指针中的数据并且返回
	if c.closed != 0 && c.qcount == 0 {
		unlock(&c.lock)
		if ep != nil {
			typedmemclr(c.elemtype, ep)
		}
		return true, false
	}

    // 等待发送队列中goroutine的存在
    // 如果goroutine存在，说明存在下面两种情况:
    // 1. 是非缓冲型的channel
    // 2. 是缓冲型的channel，但是缓冲区已经满了
    // 针对1类型：直接进行内存的拷贝，即从sender goroutine -> receiver goroutine
    // 针对2类型，接收到循环数组头部的数据，并将发送者的元素放置到循环数组尾部
	if sg := c.sendq.dequeue(); sg != nil {
		recv(c, sg, ep, func() { unlock(&c.lock) }, 3)
		return true, true
	}

    // qcount>0，说明缓冲区还有数据
	if c.qcount > 0 {
		// 直接从循环数组中找到要接收到数据
		qp := chanbuf(c, c.recvx)

        // ep不是nil，说明程序代码中的接收格式为： val <-ch 而不是 <-ch
        // 说明此时有接收对象，将chan中的数据值拷贝到ep指针对应的地方
		if ep != nil {
			typedmemmove(c.elemtype, ep, qp)
		}
        // 如果ep为nil，说明没有接收对象，那么直接将chan中的数据抹除即可
		typedmemclr(c.elemtype, qp)
        // 接收游标向前移动
		c.recvx++
        // 接收游标达到边界值，那么清除游标的值
		if c.recvx == c.dataqsiz {
			c.recvx = 0
		}
        // 数据被接收了，少了1个
		c.qcount--
        // 解锁
		unlock(&c.lock)
		return true, true
	}

    // 如果是非阻塞的情况，此时没有缓冲区，也就没有数据
	if !block {
		unlock(&c.lock)
		return false, false
	}

	// 阻塞接受，和发送类似，拿到当前 Goroutine 和 sudog 并且做一些数据填充
	gp := getg()
	mysg := acquireSudog()
	mysg.releasetime = 0
	if t0 != 0 {
		mysg.releasetime = -1
	}
	mysg.elem = ep
	mysg.waitlink = nil
	gp.waiting = mysg
	mysg.g = gp
	mysg.isSelect = false
	mysg.c = c
	gp.param = nil

    // 把 sudog 放入到接收者队列当中
	c.recvq.enqueue(mysg)
    // 然后休眠当前 Goroutine 等待唤醒
	gopark(chanparkcommit, unsafe.Pointer(&c.lock), waitReasonChanReceive, traceEvGoBlockRecv, 2)

	// Goroutine 被唤醒，接收完数据，做一些数据清理的操作，释放掉 sudog 然后返回
	if mysg != gp.waiting {
		throw("G waiting list is corrupted")
	}
	gp.waiting = nil
	gp.activeStackChans = false
	if mysg.releasetime > 0 {
		blockevent(mysg.releasetime-t0, 2)
	}
	closed := gp.param == nil
	gp.param = nil
	mysg.c = nil
	releaseSudog(mysg)
	return true, !closed
}
```

###  4.1. <a name='goroutinerecv'></a>如果有goroutine急着发送数据——#recv

```go
func recv(c *hchan, sg *sudog, ep unsafe.Pointer, unlockf func(), skip int) {
	// 如果无缓冲的 channel 直接调用 recvDirect 将数据从发送者 Goroutine 拷贝到变量
    if c.dataqsiz == 0 {
		if ep != nil {
			// copy data from sender
			recvDirect(c.elemtype, sg, ep)
		}
	} else {
		// 否则的话说明这是一个有缓冲的 channel 并且缓冲已经满了

        // 先从底层数组中拿到数据地址
		qp := chanbuf(c, c.recvx)

		// 然后把数据复制到接收变量上
		if ep != nil {
			typedmemmove(c.elemtype, ep, qp)
		}

		// 然后将发送者 Goroutine 中的数据拷贝到底层数组上
		typedmemmove(c.elemtype, qp, sg.elem)
		c.recvx++
		if c.recvx == c.dataqsiz {
			c.recvx = 0
		}
		c.sendx = c.recvx // c.sendx = (c.sendx+1) % c.dataqsiz
	}
    // 最后做一些清理操作
	sg.elem = nil
	gp := sg.g
	unlockf()
	gp.param = unsafe.Pointer(sg)
	if sg.releasetime != 0 {
		sg.releasetime = cputicks()
	}
	goready(gp, skip+1)
}
```

###  4.2. <a name='-1'></a>接收数据总结

数据接收和发送其实大同小异，也是分为检查和数据接收，数据接收又分三种情况

- 直接获取数据，如果当前有阻塞的发送者 Goroutine 走这条路
  - 如果是无缓冲 channel，直接从发送者那里把数据拷贝给接收变量
  - 如果是有缓冲 channel，并且 channel 已经满了，就先从 channel 的底层数组拷贝数据，再把阻塞的发送者 Goroutine 的数据拷贝到 channel 的循环队列中
- 从 channel 的缓冲中获取数据，有缓冲 channel 并且缓存队列有数据时走这条路
  - 直接从缓存队列中复制数据给接收变量
- 阻塞接收，剩余情况走这里
  - 和发送类似，先获取当前 Goroutine 信息，构造 sudog 加入到 channel 的 recvq 上
  - 然后休眠当前 Goroutine 等待唤醒
  - 唤醒后做一些清理工作，释放 sudog 返回

##  5. <a name='chan-1'></a>关闭chan

我们使用 `close(ch)` 来关闭 channel 最后会调用 runtime 中的 `closechan` 方法：

```go
func closechan(c *hchan) {
    // 关闭 nil 的 channel 会导致 panic
	if c == nil {
		panic(plainError("close of nil channel"))
	}

    // 加锁
	lock(&c.lock)

    // 关闭已关闭的 channel 会导致 panic
	if c.closed != 0 {
		unlock(&c.lock)
		panic(plainError("close of closed channel"))
	}

	// 设置 channel 状态
	c.closed = 1

	var glist gList

	// 释放所有的接收者 Goroutine
	for {
		sg := c.recvq.dequeue()
		if sg == nil {
			break
		}
		if sg.elem != nil {
			typedmemclr(c.elemtype, sg.elem)
			sg.elem = nil
		}
		if sg.releasetime != 0 {
			sg.releasetime = cputicks()
		}
		gp := sg.g
		gp.param = nil

		glist.push(gp)
	}

	// 释放所有的发送者channel，会 panic 因为不允许向已关闭的 channel 发送数据
	for {
		sg := c.sendq.dequeue()
		if sg == nil {
			break
		}
		sg.elem = nil
		if sg.releasetime != 0 {
			sg.releasetime = cputicks()
		}
		gp := sg.g
		gp.param = nil
		if raceenabled {
			raceacquireg(gp, c.raceaddr())
		}
		glist.push(gp)
	}
	unlock(&c.lock)

	// 将所有的 Goroutine 设置为可运行状态
	for !glist.empty() {
		gp := glist.pop()
		gp.schedlink = 0
		goready(gp, 3)
	}
}
```

###  5.1. <a name='chan-1'></a>关闭chan总结

- 关闭一个 nil 的 channel 和已关闭了的 channel 都会导致 panic
- 关闭 channel 后会释放所有因为 channel 而阻塞的 Goroutine

## 案例说明(好案例)

```go
func gA(a <-chan int) {
    val := <-a
    fmt.Println("g1 received data:", val)
    return
}

func gB(b <-chan int) {
    val := <-b
    fmt.Println("g2 received data:", val)
    return
}

func main() {
    ch := make(chan int)
    go gA(ch)
    go gB(ch)
    ch <- 3
    time.Sleep(time.Second)
}
```

首先创建了一个无缓冲的 channel，接着启动两个 goroutine，并将前面创建的 channel 传递进 去。然后，向这个 channel 中发送数据 3，最后 sleep 1 秒后程序退出。

## 问题

### 如果channel为nil，从这个channel中接收数据会怎么样？

* 在非阻塞模式下，会直接返回
* 在阻塞模式下，会调用`gopark`挂起goroutine，并且会一直阻塞下去

## 关闭一个为nil的channel会怎么样?

关闭一个nil的channel或者一个已经关闭的channel都会导致panic

## channel什么情况下会引起资源的泄露?

channel资源泄露的原因是goroutine操作channel之后，处于发送或者接收阻塞状态，而channel处于满或者空的状态，一直得不到改变。他同时，垃圾回收器也不会回收这部分的资源，就会导致goroutine一直处于等待的状态。

### channel何时被GC收集?

如果一个channel没有任何goroutine引用，GC就会对其进行回收操作。
