# Go源码阅读——sync.waitGroup

<!-- vscode-markdown-toc -->
* 1. [前言](#)
* 2. [waitGroup结构体定义](#waitGroup)
	* 2.1. [为什么state设计成12字节的类型?](#state12)
* 3. [#Add](#Add)
* 4. [#Wait](#Wait)
* 5. [#Done](#Done)
* 6. [问题](#-1)
	* 6.1. [waitGroup支持一个goroutine等待多个goroutine执行完成吗?](#waitGroupgoroutinegoroutine)
	* 6.2. [waitGroup支持多个goroutine等待一个goroutine吗?](#waitGroupgoroutinegoroutine-1)

<!-- vscode-markdown-toc-config
	numbering=true
	autoSave=true
	/vscode-markdown-toc-config -->
<!-- /vscode-markdown-toc -->

##  1. <a name=''></a>前言

`WaitGroup` 可以解决一个 goroutine 等待多个 goroutine 同时结束的场景，这个比较常见的场景就是例如 后端 worker 启动了多个消费者干活，还有爬虫并发爬取数据，多线程下载等等。

那么在这里抛出一个问题：**支持多个goroutine等待一个goroutine吗？** 。这个问题先不回答，会放在下面的源码分析中解答。

##  2. <a name='waitGroup'></a>waitGroup结构体定义

```go
type WaitGroup struct {
	noCopy noCopy

	// 64-bit value: high 32 bits are counter, low 32 bits are waiter count.
	// 64-bit atomic operations require 64-bit alignment, but 32-bit
	// compilers do not ensure it. So we allocate 12 bytes and then use
	// the aligned 8 bytes in them as state, and the other 4 as storage
	// for the sema.
	state1 [3]uint32
}
```

`WaitGroup` 结构十分简单，由 `nocopy` 和 `state1` 两个字段组成，其中 `nocopy` 是用来防止复制的。由于嵌入了 `nocopy` 所以在执行 `go vet` 时如果检查到 `WaitGroup` 被复制了就会报错。这样可以一定程度上保证 `WaitGroup` 不被复制，对了直接 go run 是不会有错误的，所以我们代码 push 之前都会强制要求进行 lint 检查，在 ci/cd 阶段也需要先进行 lint 检查，避免出现这种类似的错误。

`state1` 的设计非常巧妙，这是一个是十二字节的数据，这里面主要包含两大块，counter 占用了 8 字节用于计数，sema 占用 4 字节用做信号量。

###  2.1. <a name='state12'></a>为什么state设计成12字节的类型?

在做 64 位的原子操作的时候必须要保证 64 位（8 字节）对齐，如果没有对齐的就会有问题，但是 32 位的编译器并不能保证 64 位对齐所以这里用一个 12 字节的 `state1`字段来存储这两个状态，然后根据是否 8 字节对齐选择不同的保存方式。

<div align=center><img src="/assets/wg1.svg"/></div>

这个操作巧妙在哪里呢？

- 如果是 64 位的机器那肯定是 8 字节对齐了的，所以是上面第一种方式
- 如果在 32 位的机器上
  - 如果恰好 8 字节对齐了，那么也是第一种方式取前面的 8 字节数据
  - 如果是没有对齐，但是 32 位 4 字节是对齐了的，所以我们只需要后移四个字节，那么就 8 字节对齐了，所以是第二种方式

**所以通过 sema 信号量这四个字节的位置不同，保证了 counter 这个字段无论在 32 位还是 64 为机器上都是 8 字节对齐的，后续做 64 位原子操作的时候就没问题了** 。

```go
func (wg *WaitGroup) state() (statep *uint64, semap *uint32) {
	if uintptr(unsafe.Pointer(&wg.state1))%8 == 0 {
		return (*uint64)(unsafe.Pointer(&wg.state1)), &wg.state1[2]
	} else {
		return (*uint64)(unsafe.Pointer(&wg.state1[1])), &wg.state1[0]
	}
}
```

上面的代码会根据对齐类型来返回对应的`counter`和`sema` 。

##  3. <a name='Add'></a>#Add

```go
func (wg *WaitGroup) Add(delta int) {
    // 先从 state 当中把数据和信号量取出来
	statep, semap := wg.state()

    // 在 waiter 上加上 delta 值
	state := atomic.AddUint64(statep, uint64(delta)<<32)
    // 取出当前的 counter
	v := int32(state >> 32)
    // 取出当前的 waiter，正在等待 goroutine 数量
	w := uint32(state)

    // counter 不能为负数
	if v < 0 {
		panic("sync: negative WaitGroup counter")
	}

    // 这里属于防御性编程
    // w != 0 说明现在已经有 goroutine 在等待中，说明已经调用了 Wait() 方法
    // 这时候 delta > 0 && v == int32(delta) 说明在调用了 Wait() 方法之后又想加入新的等待者
    // 这种操作是不允许的
	if w != 0 && delta > 0 && v == int32(delta) {
		panic("sync: WaitGroup misuse: Add called concurrently with Wait")
	}
    // 如果当前没有人在等待就直接返回，并且 counter > 0
	if v > 0 || w == 0 {
		return
	}

    // 这里也是防御 主要避免并发调用 add 和 wait
	if *statep != state {
		panic("sync: WaitGroup misuse: Add called concurrently with Wait")
	}

	// 唤醒所有 waiter
	*statep = 0
	for ; w != 0; w-- {
		runtime_Semrelease(semap, false, 0)
	}
}
```

##  4. <a name='Wait'></a>#Wait

wait 主要就是等待其他的 goroutine 完事之后唤醒

```go
func (wg *WaitGroup) Wait() {
	// 先从 state 当中把数据和信号量的地址取出来
    statep, semap := wg.state()

	for {
     	// 这里去除 counter 和 waiter 的数据
		state := atomic.LoadUint64(statep)
		v := int32(state >> 32)
		w := uint32(state)

        // counter = 0 说明没有在等的，直接返回就行
        if v == 0 {
			// Counter is 0, no need to wait.
			return
		}

		// waiter + 1，调用一次就多一个等待者，然后休眠当前 goroutine 等待被唤醒
		if atomic.CompareAndSwapUint64(statep, state, state+1) {
			runtime_Semacquire(semap)
			if *statep != 0 {
				panic("sync: WaitGroup is reused before previous Wait has returned")
			}
			return
		}
	}
}
```

##  5. <a name='Done'></a>#Done

```go
func (wg *WaitGroup) Done() {
	wg.Add(-1)
}
```

##  6. <a name='-1'></a>问题

###  6.1. <a name='waitGroupgoroutinegoroutine'></a>waitGroup支持一个goroutine等待多个goroutine执行完成吗?

支持。只需要一个goroutine执行`Wait`，其他负责`Done`即可。

###  6.2. <a name='waitGroupgoroutinegoroutine-1'></a>waitGroup支持多个goroutine等待一个goroutine吗?

支持。当执行了`#Add`之后，会唤醒所有的等待goroutine。

```go
// #Add	
	// 唤醒所有 waiter
	*statep = 0
	for ; w != 0; w-- {
		runtime_Semrelease(semap, false, 0)
	}
```

