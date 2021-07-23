# Go语言通道Channel

<!-- vscode-markdown-toc -->
* 1. [前言](#)
* 2. [Channel设计原理](#Channel)
* 3. [Channel的数据结构](#Channel-1)
* 4. [创建Channel](#Channel-1)
* 5. [发送数据](#-1)
	* 5.1. [直接发送](#-1)
	* 5.2. [缓冲区](#-1)
	* 5.3. [阻塞发送](#-1)
* 6. [接收数据](#-1)
	* 6.1. [直接接收](#-1)
	* 6.2. [缓冲区](#-1)
	* 6.3. [阻塞接收](#-1)

<!-- vscode-markdown-toc-config
	numbering=true
	autoSave=true
	/vscode-markdown-toc-config -->
<!-- /vscode-markdown-toc -->

##  1. <a name=''></a>前言

Channel是Go的核心数据结构和Goroutine之间的通信方式，Channel是支撑Go语言高性能并发编程模型的重要结构。

##  2. <a name='Channel'></a>Channel设计原理

Go语言中最常见的，也是经常被人提及的设计模式是：**不要通过共享内存的方式进行通信，要通过通信的方式共享内存** 。很多主流的编程语言中，多个线程传递数据的方式一般都是共享内存，为了解决线程的竞争，我们需要限制同一时间能够读写这些变量的线程数量，**然而这与Go语言鼓励的设计并不相同** 。

Go语言虽然也能通过共享内存的方式进行通信，但是Go语言提供了一种不同的并发模型，即通信顺序进程(Communicating Sequential Processes, CSP)。Goroutine和Channel分别对应CSP中的实体和传递信息的媒介，**Goroutine之间通过Channel传递数据** 。

在日常的并发中，我们总是将锁分为乐观锁和悲观锁，**乐观锁并不是一种真正的锁，像Mutex，而是一种并发控制的思想** 。

乐观锁的本质是**基于验证的协议** ，使用原子指令CAS在多线程中同步数据，无锁队列的实现也依赖于这一原子指令。

Channel在运行时的内部表示是[`runtime.hchan`]() ，该结构体包含了用于保护成员变量的互斥锁，从某种程度来说，**Channel是一个用于同步和通信的有锁队列，使用互斥锁解决程序中可能存在的线程竞争问题** 。然而锁的休眠和唤醒会带来额外的上下文切换，如果临界区过大，加锁解锁会导致额外开销，此时就会称为性能瓶颈。

* 同步Channel——不需要缓冲区，发送方会直接将数据交给(HandOff)接收方
* 异步Channel——基于环形缓存的传统生产者消费者模型
* [`chan struct{}`]() 类型的异步Channel——[`struct{}`]()类型不占用内存空间，不需要实现缓冲区和直接发送(HandOff)的语义

##  3. <a name='Channel-1'></a>Channel的数据结构

Go 语言的 Channel 在运行时使用 [`runtime.hchan`](https://draveness.me/golang/tree/runtime.hchan) 结构体表示。我们在 Go 语言中创建新的 Channel 时，实际上创建的都是如下所示的结构：

```go
type hchan struct {
	qcount   uint
	dataqsiz uint
	buf      unsafe.Pointer
	elemsize uint16
	closed   uint32
	elemtype *_type
	sendx    uint
	recvx    uint
	recvq    waitq
	sendq    waitq

	lock mutex
}
```

[`runtime.hchan`](https://draveness.me/golang/tree/runtime.hchan) 结构体中的五个字段 `qcount`、`dataqsiz`、`buf`、`sendx`、`recv` 构建底层的循环队列：

- `qcount` — Channel 中的元素个数；
- `dataqsiz` — Channel 中的循环队列的长度；
- `buf` — Channel 的缓冲区数据指针；
- `sendx` — Channel 的发送操作处理到的位置；
- `recvx` — Channel 的接收操作处理到的位置；

除此之外，`elemsize` 和 `elemtype` 分别表示当前 Channel 能够收发的元素类型和大小；`sendq` 和 `recvq` 存储了当前 Channel 由于缓冲区空间不足而阻塞的 Goroutine 列表，这些等待队列使用双向链表 [`runtime.waitq`](https://draveness.me/golang/tree/runtime.waitq) 表示，链表中所有的元素都是 [`runtime.sudog`](https://draveness.me/golang/tree/runtime.sudog) 结构：

```go
type waitq struct {
	first *sudog
	last  *sudog
}
```

[`runtime.sudog`](https://draveness.me/golang/tree/runtime.sudog) 表示一个在等待列表中的 Goroutine，该结构中存储了两个分别指向前后 [`runtime.sudog`](https://draveness.me/golang/tree/runtime.sudog) 的指针以构成链表。

##  4. <a name='Channel-1'></a>创建Channel

Go 语言中所有 Channel 的创建都会使用 `make` 关键字。编译器会将 `make(chan int, 10)` 表达式转换成 `OMAKE` 类型的节点，并在类型检查阶段将 `OMAKE` 类型的节点转换成 `OMAKECHAN` 类型：

```go
func typecheck1(n *Node, top int) (res *Node) {
	switch n.Op {
	case OMAKE:
		...
		switch t.Etype {
		case TCHAN:
			l = nil
			if i < len(args) { // 带缓冲区的异步 Channel
				...
				n.Left = l
			} else { // 不带缓冲区的同步 Channel
				n.Left = nodintconst(0)
			}
			n.Op = OMAKECHAN
		}
	}
}
```

这一阶段会对传入 `make` 关键字的缓冲区大小进行检查，如果我们不向 `make` 传递表示缓冲区大小的参数，那么就会设置一个默认值 0，也就是当前的 Channel 不存在缓冲区。

`OMAKECHAN` 类型的节点最终都会在 SSA 中间代码生成阶段之前被转换成调用 [`runtime.makechan`](https://draveness.me/golang/tree/runtime.makechan) 或者 [`runtime.makechan64`](https://draveness.me/golang/tree/runtime.makechan64) 的函数：

这一阶段会对传入 `make` 关键字的缓冲区大小进行检查，**如果我们不向 `make` 传递表示缓冲区大小的参数，那么就会设置一个默认值 0，也就是当前的 Channel 不存在缓冲区**。

`OMAKECHAN` 类型的节点最终都会在 SSA 中间代码生成阶段之前被转换成调用 [`runtime.makechan`](https://draveness.me/golang/tree/runtime.makechan) 或者 [`runtime.makechan64`](https://draveness.me/golang/tree/runtime.makechan64) 的函数：

```go
func walkexpr(n *Node, init *Nodes) *Node {
	switch n.Op {
	case OMAKECHAN:
		size := n.Left
		fnname := "makechan64"
		argtype := types.Types[TINT64]

		if size.Type.IsKind(TIDEAL) || maxintval[size.Type.Etype].Cmp(maxintval[TUINT]) <= 0 {
			fnname = "makechan"
			argtype = types.Types[TINT]
		}
		n = mkcall1(chanfn(fnname, 1, n.Type), n.Type, init, typename(n.Type), conv(size, argtype))
	}
}
```

[`runtime.makechan`](https://draveness.me/golang/tree/runtime.makechan) 和 [`runtime.makechan64`](https://draveness.me/golang/tree/runtime.makechan64) 会根据传入的参数类型和缓冲区大小创建一个新的 Channel 结构，**其中后者用于处理缓冲区大小大于 2 的 32 次方的情况**，因为这在 Channel 中并不常见，所以我们重点关注 [`runtime.makechan`](https://draveness.me/golang/tree/runtime.makechan)：

```go
func makechan(t *chantype, size int) *hchan {
	elem := t.elem
	mem, _ := math.MulUintptr(elem.size, uintptr(size))

	var c *hchan
	switch {
	case mem == 0:
		c = (*hchan)(mallocgc(hchanSize, nil, true))
		c.buf = c.raceaddr()
	case elem.kind&kindNoPointers != 0:
		c = (*hchan)(mallocgc(hchanSize+mem, nil, true))
		c.buf = add(unsafe.Pointer(c), hchanSize)
	default:
		c = new(hchan)
		c.buf = mallocgc(mem, elem, true)
	}
	c.elemsize = uint16(elem.size)
	c.elemtype = elem
	c.dataqsiz = uint(size)
	return c
}
```

上述代码根据 Channel 中收发元素的类型和缓冲区的大小初始化 [`runtime.hchan`](https://draveness.me/golang/tree/runtime.hchan) 和缓冲区：

- 如果当前 Channel 中不存在缓冲区，那么就只会为 [`runtime.hchan`](https://draveness.me/golang/tree/runtime.hchan) 分配一段内存空间；
- 如果当前 Channel 中存储的类型不是指针类型，会为当前的 Channel 和底层的数组分配一块连续的内存空间；
- 在默认情况下会单独为 [`runtime.hchan`](https://draveness.me/golang/tree/runtime.hchan) 和缓冲区分配内存；

在函数的最后会统一更新 [`runtime.hchan`](https://draveness.me/golang/tree/runtime.hchan) 的 `elemsize`、`elemtype` 和 `dataqsiz` 几个字段。

##  5. <a name='-1'></a>发送数据

当我们想要向 Channel 发送数据时，就需要使用 `ch <- i` 语句，编译器会将它解析成 `OSEND` 节点并在 [`cmd/compile/internal/gc.walkexpr`](https://draveness.me/golang/tree/cmd/compile/internal/gc.walkexpr) 中转换成 [`runtime.chansend1`](https://draveness.me/golang/tree/runtime.chansend1)：

```go
func walkexpr(n *Node, init *Nodes) *Node {
	switch n.Op {
	case OSEND:
		n1 := n.Right
		n1 = assignconv(n1, n.Left.Type.Elem(), "chan send")
		n1 = walkexpr(n1, init)
		n1 = nod(OADDR, n1, nil)
		n = mkcall1(chanfn("chansend1", 2, n.Left.Type), nil, init, n.Left, n1)
	}
}
```

[`runtime.chansend1`](https://draveness.me/golang/tree/runtime.chansend1) 只是调用了 [`runtime.chansend`](https://draveness.me/golang/tree/runtime.chansend) 并传入 Channel 和需要发送的数据。[`runtime.chansend`](https://draveness.me/golang/tree/runtime.chansend) 是向 Channel 中发送数据时一定会调用的函数，该函数包含了发送数据的全部逻辑，如果我们在调用时将 `block` 参数设置成 `true`，那么表示当前发送操作是阻塞的：

```go
func chansend(c *hchan, ep unsafe.Pointer, block bool, callerpc uintptr) bool {
	lock(&c.lock)

	if c.closed != 0 {
		unlock(&c.lock)
		panic(plainError("send on closed channel"))
	}
```

**在发送数据的逻辑执行之前会先为当前 Channel 加锁，防止多个线程并发修改数据**。如果 Channel 已经关闭，那么向该 Channel 发送数据时会报 “send on closed channel” 错误并中止程序。

**因为 [`runtime.chansend`](https://draveness.me/golang/tree/runtime.chansend) 函数的实现比较复杂，所以我们这里将该函数的执行过程分成以下的三个部分：**

- 当存在等待的接收者时，通过 [`runtime.send`](https://draveness.me/golang/tree/runtime.send) 直接将数据发送给阻塞的接收者；
- 当缓冲区存在空余空间时，将发送的数据写入 Channel 的缓冲区；
- 当不存在缓冲区或者缓冲区已满时，等待其他 Goroutine 从 Channel 接收数据；

###  5.1. <a name='-1'></a>直接发送

如果目标 Channel 没有被关闭并且已经有处于读等待的 Goroutine，那么 [`runtime.chansend`](https://draveness.me/golang/tree/runtime.chansend) 会从接收队列 `recvq` 中取出**最先陷入等待的 Goroutine** 并直接向它发送数据：

```go
	if sg := c.recvq.dequeue(); sg != nil {
		send(c, sg, ep, func() { unlock(&c.lock) }, 3)
		return true
	}
```

下图展示了 Channel 中存在等待数据的 Goroutine 时，向 Channel 发送数据的过程：

<div align=center><img src="../assets/channel1.png"/>></div>

发送数据时会调用 [`runtime.send`](https://draveness.me/golang/tree/runtime.send)，该函数的执行可以分成两个部分：

1. 调用 [`runtime.sendDirect`](https://draveness.me/golang/tree/runtime.sendDirect) 将发送的数据直接拷贝到 `x = <-c` 表达式中变量 `x` 所在的内存地址上；
2. 调用 [`runtime.goready`](https://draveness.me/golang/tree/runtime.goready) 将等待接收数据的 Goroutine 标记成可运行状态 `Grunnable` 并把该 Goroutine 放到发送方所在的处理器的 `runnext` 上等待执行，该处理器在下一次调度时会立刻唤醒数据的接收方；

```go
func send(c *hchan, sg *sudog, ep unsafe.Pointer, unlockf func(), skip int) {
	if sg.elem != nil {
		sendDirect(c.elemtype, sg, ep)
		sg.elem = nil
	}
	gp := sg.g
	unlockf()
	gp.param = unsafe.Pointer(sg)
	goready(gp, skip+1)
}
```

需要注意的是，**发送数据的过程只是将接收方的 Goroutine 放到了处理器的 `runnext` 中，程序没有立刻执行该 Goroutine**。

###  5.2. <a name='-1'></a>缓冲区

如果创建的 Channel 包含缓冲区并且 Channel 中的数据没有装满，会执行下面这段代码：

```go
func chansend(c *hchan, ep unsafe.Pointer, block bool, callerpc uintptr) bool {
	...
	if c.qcount < c.dataqsiz {
		qp := chanbuf(c, c.sendx)
		typedmemmove(c.elemtype, qp, ep)
		c.sendx++
		if c.sendx == c.dataqsiz {
			c.sendx = 0
		}
		c.qcount++
		unlock(&c.lock)
		return true
	}
	...
}
```

在这里我们首先会使用 [`runtime.chanbuf`](https://draveness.me/golang/tree/runtime.chanbuf) 计算出下一个可以存储数据的位置，然后通过 [`runtime.typedmemmove`](https://draveness.me/golang/tree/runtime.typedmemmove) 将发送的数据拷贝到缓冲区中并增加 `sendx` 索引和 `qcount` 计数器。

<div align=center><img src="../assets/channel2.png"/></div>

如果当前 Channel 的缓冲区未满，向 Channel 发送的数据会存储在 Channel 的 `sendx` 索引所在的位置并将 `sendx` 索引加一。因为这里的 `buf` 是一个循环数组，所以当 `sendx` 等于 `dataqsiz` 时会重新回到数组开始的位置。

###  5.3. <a name='-1'></a>阻塞发送

当 Channel 没有接收者能够处理数据时，向 Channel 发送数据会被下游阻塞，当然使用 `select` 关键字可以向 Channel 非阻塞地发送消息。向 Channel 阻塞地发送数据会执行下面的代码，我们可以简单梳理一下这段代码的逻辑：

```go
func chansend(c *hchan, ep unsafe.Pointer, block bool, callerpc uintptr) bool {
	...
	if !block {
		unlock(&c.lock)
		return false
	}

	gp := getg()
	mysg := acquireSudog()
	mysg.elem = ep
	mysg.g = gp
	mysg.c = c
	gp.waiting = mysg
	c.sendq.enqueue(mysg)
	goparkunlock(&c.lock, waitReasonChanSend, traceEvGoBlockSend, 3)

	gp.waiting = nil
	gp.param = nil
	mysg.c = nil
	releaseSudog(mysg)
	return true
}
```

1. 调用 [`runtime.getg`](https://draveness.me/golang/tree/runtime.getg) 获取发送数据使用的 Goroutine；
2. 执行 [`runtime.acquireSudog`](https://draveness.me/golang/tree/runtime.acquireSudog) 获取 [`runtime.sudog`](https://draveness.me/golang/tree/runtime.sudog) 结构并设置这一次阻塞发送的相关信息，例如发送的 Channel、是否在 select 中和待发送数据的内存地址等；
3. 将刚刚创建并初始化的 [`runtime.sudog`](https://draveness.me/golang/tree/runtime.sudog) 加入发送等待队列，并设置到当前 Goroutine 的 `waiting` 上，表示 Goroutine 正在等待该 `sudog` 准备就绪；
4. 调用 [`runtime.goparkunlock`](https://draveness.me/golang/tree/runtime.goparkunlock) 将当前的 Goroutine 陷入沉睡等待唤醒；
5. 被调度器唤醒后会执行一些收尾工作，将一些属性置零并且释放 [`runtime.sudog`](https://draveness.me/golang/tree/runtime.sudog) 结构体；

函数在最后会返回 `true` 表示这次我们已经成功向 Channel 发送了数据。

##  6. <a name='-1'></a>接收数据

我们接下来继续介绍 Channel 操作的另一方：接收数据。Go 语言中可以使用两种不同的方式去接收 Channel 中的数据：

```go
i <- ch
i, ok <- ch
```

这两种不同的方法经过编译器的处理都会变成 `ORECV` 类型的节点，后者会在类型检查阶段被转换成 `OAS2RECV` 类型。数据的接收操作遵循以下的路线图：

<div align=center><img src="../assets/channel3.png"/></div>

虽然不同的接收方式会被转换成 [`runtime.chanrecv1`](https://draveness.me/golang/tree/runtime.chanrecv1) 和 [`runtime.chanrecv2`](https://draveness.me/golang/tree/runtime.chanrecv2) 两种不同函数的调用，但是这两个函数最终还是会调用 [`runtime.chanrecv`](https://draveness.me/golang/tree/runtime.chanrecv)。

当我们从一个空 Channel 接收数据时会直接调用 [`runtime.gopark`](https://draveness.me/golang/tree/runtime.gopark) 让出处理器的使用权。

```go
func chanrecv(c *hchan, ep unsafe.Pointer, block bool) (selected, received bool) {
	if c == nil {
		if !block {
			return
		}
		gopark(nil, nil, waitReasonChanReceiveNilChan, traceEvGoStop, 2)
		throw("unreachable")
	}

	lock(&c.lock)

	if c.closed != 0 && c.qcount == 0 {
		unlock(&c.lock)
		if ep != nil {
			typedmemclr(c.elemtype, ep)
		}
		return true, false
	}
```

如果当前 Channel 已经被关闭并且缓冲区中不存在任何数据，那么会清除 `ep` 指针中的数据并立刻返回。

除了上述两种特殊情况，使用 [`runtime.chanrecv`](https://draveness.me/golang/tree/runtime.chanrecv) 从 Channel 接收数据时还包含以下三种不同情况：

- 当存在等待的发送者时，通过 [`runtime.recv`](https://draveness.me/golang/tree/runtime.recv) 从阻塞的发送者或者缓冲区中获取数据；
- 当缓冲区存在数据时，从 Channel 的缓冲区中接收数据；
- 当缓冲区中不存在数据时，等待其他 Goroutine 向 Channel 发送数据；

###  6.1. <a name='-1'></a>直接接收

当 Channel 的 `sendq` 队列中包含处于等待状态的 Goroutine 时，该函数会取出队列头等待的 Goroutine，处理的逻辑和发送时相差无几，只是发送数据时调用的是 [`runtime.send`](https://draveness.me/golang/tree/runtime.send) 函数，而接收数据时使用 [`runtime.recv`](https://draveness.me/golang/tree/runtime.recv)：

```go
if sg := c.sendq.dequeue(); sg != nil {
		recv(c, sg, ep, func() { unlock(&c.lock) }, 3)
		return true, true
	}
```

[`runtime.recv`](https://draveness.me/golang/tree/runtime.recv) 的实现比较复杂：

```go
func recv(c *hchan, sg *sudog, ep unsafe.Pointer, unlockf func(), skip int) {
	if c.dataqsiz == 0 {
		if ep != nil {
			recvDirect(c.elemtype, sg, ep)
		}
	} else {
		qp := chanbuf(c, c.recvx)
		if ep != nil {
			typedmemmove(c.elemtype, ep, qp)
		}
		typedmemmove(c.elemtype, qp, sg.elem)
		c.recvx++
		c.sendx = c.recvx // c.sendx = (c.sendx+1) % c.dataqsiz
	}
	gp := sg.g
	gp.param = unsafe.Pointer(sg)
	goready(gp, skip+1)
}
```

该函数会根据缓冲区的大小分别处理不同的情况：

- 如果 Channel 不存在缓冲区；
  1. 调用 [`runtime.recvDirect`](https://draveness.me/golang/tree/runtime.recvDirect) 将 Channel 发送队列中 Goroutine 存储的 `elem` 数据拷贝到目标内存地址中；
- 如果 Channel 存在缓冲区；
  1. 将队列中的数据拷贝到接收方的内存地址；
  2. 将发送队列头的数据拷贝到缓冲区中，释放一个阻塞的发送方；

无论发生哪种情况，运行时都会调用 [`runtime.goready`](https://draveness.me/golang/tree/runtime.goready) 将当前处理器的 `runnext` 设置成发送数据的 Goroutine，在调度器下一次调度时将阻塞的发送方唤醒。

<div align=center><img src="../assets/channel4.png"/></div>

上图展示了 Channel 在缓冲区已经没有空间并且发送队列中存在等待的 Goroutine 时，运行 `<-ch` 的执行过程。发送队列头的 [`runtime.sudog`](https://draveness.me/golang/tree/runtime.sudog) 中的元素会替换接收索引 `recvx` 所在位置的元素，原有的元素会被拷贝到接收数据的变量对应的内存空间上。

###  6.2. <a name='-1'></a>缓冲区

当 Channel 的缓冲区中已经包含数据时，从 Channel 中接收数据会直接从缓冲区中 `recvx` 的索引位置中取出数据进行处理：

```go
func chanrecv(c *hchan, ep unsafe.Pointer, block bool) (selected, received bool) {
	...
	if c.qcount > 0 {
		qp := chanbuf(c, c.recvx)
		if ep != nil {
			typedmemmove(c.elemtype, ep, qp)
		}
		typedmemclr(c.elemtype, qp)
		c.recvx++
		if c.recvx == c.dataqsiz {
			c.recvx = 0
		}
		c.qcount--
		return true, true
	}
	...
}
```

如果接收数据的内存地址不为空，那么会使用 [`runtime.typedmemmove`](https://draveness.me/golang/tree/runtime.typedmemmove) 将缓冲区中的数据拷贝到内存中、清除队列中的数据并完成收尾工作。

收尾工作包括递增 `recvx`，一旦发现索引超过了 Channel 的容量时，会将它归零重置循环队列的索引；除此之外，该函数还会减少 `qcount` 计数器并释放持有 Channel 的锁。

###  6.3. <a name='-1'></a>阻塞接收

当 Channel 的发送队列中不存在等待的 Goroutine 并且缓冲区中也不存在任何数据时，从管道中接收数据的操作会变成阻塞的，然而不是所有的接收操作都是阻塞的，与 `select` 语句结合使用时就可能会使用到非阻塞的接收操作：

```go
func chanrecv(c *hchan, ep unsafe.Pointer, block bool) (selected, received bool) {
	...
	if !block {
		unlock(&c.lock)
		return false, false
	}

	gp := getg()
	mysg := acquireSudog()
	mysg.elem = ep
	gp.waiting = mysg
	mysg.g = gp
	mysg.c = c
	c.recvq.enqueue(mysg)
	goparkunlock(&c.lock, waitReasonChanReceive, traceEvGoBlockRecv, 3)

	gp.waiting = nil
	closed := gp.param == nil
	gp.param = nil
	releaseSudog(mysg)
	return true, !closed
}
```

在正常的接收场景中，我们会使用 [`runtime.sudog`](https://draveness.me/golang/tree/runtime.sudog) 将当前 Goroutine 包装成一个处于等待状态的 Goroutine 并将其加入到接收队列中。

完成入队之后，上述代码还会调用 [`runtime.goparkunlock`](https://draveness.me/golang/tree/runtime.goparkunlock) 立刻触发 Goroutine 的调度，让出处理器的使用权并等待调度器的调度。

