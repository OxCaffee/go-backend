# Golang runtime包之调度器源码proc

<!-- vscode-markdown-toc -->
* 1. [前言](#)
* 2. [proc.go中的文档](#proc.go)
	* 2.1. [M的parking和unparking](#Mparkingunparking)
	* 2.2. [Goroutine的状态表](#Goroutine)
* 3. [启动调度器](#-1)
	* 3.1. [创建Goroutine](#Goroutine-1)
	* 3.2. [Goroutine 结构体的创建过程](#Goroutine-1)
	* 3.3. [初始化Goroutine结构体](#Goroutine-1)
	* 3.4. [设置Goroutine调度信息](#Goroutine-1)

<!-- vscode-markdown-toc-config
	numbering=true
	autoSave=true
	/vscode-markdown-toc-config -->
<!-- /vscode-markdown-toc -->

##  1. <a name=''></a>前言

Golang的调度器源码位于`src/runtime/proc.go`中，其中关于GMP模型的`g, m, p`定义于`src/runtime/runtime2.go`中。在`proc.go`的源码中，官方给出了调度的一些文档，下面就先来看一下其中的文档。

##  2. <a name='proc.go'></a>proc.go中的文档

###  2.1. <a name='Mparkingunparking'></a>M的parking和unparking

1. **为什么需要对M进行parking和unparking?**

简而言之就是为了控制工作线程(`working thread`)的数量在一个合理的区间，即取得一个平衡(`balance`)，既不影响硬件的并行性能，又很好地保留CPU资源。

2. **对M进行parking和unparking有哪些困难?**

* 调度器(`scheduler`)的调度机制是完全并发性的，每个P都有自己的工作队列`work queue`，因此无法去计算出全局的状态。
* 为了提高调度效率，我们往往需要知道**未来发生的一些事情(future)** ，例如，不要去挂起一个即将`ready`的goroutine所在的M。

3. **golang过去的一些错误做法** 

* 中心化调度器的状态，这种做法导致前期的调度器性能非常差，并且不利于扩展。
* 直接进行goroutine的切换(`handoff`)，即当我们准备好一个goroutine之后，并且有一个空闲的P，将它和一个M直接交给该P，这将会导致M的**抖动(thrashing)** ，因为goroutine可能很快完成它的工作，那么此时我们就需要挂起M，这会引入几个问题：
  * 破坏计算的局部性，因为我们想保留本地计算的一些结果，也就是我们希望一个goroutine能够尽可能依赖一个M，好保留之前计算的一些结果，如果直接`handoff`，就会产生这样的困难
  * 引入多余的延迟`lantency`

4. **golang目前的parking和unparking策略**

golang目前只会在下面情况去挂起一个M: **存在空闲idle的P，并且没有自旋spinning的M**

###  2.2. <a name='Goroutine'></a>Goroutine的状态表

| 状态          | 描述                                                         |
| ------------- | ------------------------------------------------------------ |
| `_Gidle`      | 刚刚被分配并且还没有被初始化                                 |
| `_Grunnable`  | 没有执行代码，没有栈的所有权，存储在运行队列中               |
| `_Grunning`   | 可以执行代码，拥有栈的所有权，被赋予了内核线程 M 和处理器 P  |
| `_Gsyscall`   | 正在执行系统调用，拥有栈的所有权，没有执行用户代码，被赋予了内核线程 M 但是不在运行队列上 |
| `_Gwaiting`   | 由于运行时而被阻塞，没有执行用户代码并且不在运行队列上，但是可能存在于 Channel 的等待队列上 |
| `_Gdead`      | 没有被使用，没有执行代码，可能有分配的栈                     |
| `_Gcopystack` | 栈正在被拷贝，没有执行代码，不在运行队列上                   |
| `_Gpreempted` | 由于抢占而被阻塞，没有执行用户代码并且不在运行队列上，等待唤醒 |
| `_Gscan`      | GC 正在扫描栈空间，没有执行代码，可以与其他状态同时存在      |

##  3. <a name='-1'></a>启动调度器

调度器的启动过程是我们平时比较难以接触的过程，不过作为程序启动前的准备工作，理解调度器的启动过程对我们理解调度器的实现原理很有帮助，运行时通过 `runtime.schedinit()` 初始化调度器：

```go
func schedinit() {
	_g_ := getg()
	...

	sched.maxmcount = 10000

	...
	sched.lastpoll = uint64(nanotime())
	procs := ncpu
	if n, ok := atoi32(gogetenv("GOMAXPROCS")); ok && n > 0 {
		procs = n
	}
	if procresize(procs) != nil {
		throw("unknown runnable goroutine during bootstrap")
	}
}
```

在调度器初始函数执行的过程中会将 `maxmcount` 设置成 10000，这也就是一个 Go 语言程序能够创建的最大线程数，虽然最多可以创建 10000 个线程，但是可以同时运行的线程还是由 `GOMAXPROCS` 变量控制。

我们从环境变量 `GOMAXPROCS` 获取了程序能够同时运行的最大处理器数之后就会调用 `runtime.procresize`更新程序中处理器的数量，在这时整个程序不会执行任何用户 Goroutine，调度器也会进入锁定状态，`runtime.procresize`的执行过程如下：

1. 如果全局变量 `allp` 切片中的处理器数量少于期望数量，会对切片进行扩容；
2. 使用 `new` 创建新的处理器结构体并调用 `runtime.p.init`初始化刚刚扩容的处理器；
3. 通过指针将线程 m0 和处理器 `allp[0]` 绑定到一起；
4. 调用 `runtime.p.destroy`释放不再使用的处理器结构；
5. 通过截断改变全局变量 `allp` 的长度保证与期望处理器数量相等；
6. 将除 `allp[0]` 之外的处理器 P 全部设置成 `_Pidle` 并加入到全局的空闲队列中；

调用 `runtime.procresize`是调度器启动的最后一步，在这一步过后调度器会完成相应数量处理器的启动，等待用户创建运行新的 Goroutine 并为 Goroutine 调度处理器资源。

###  3.1. <a name='Goroutine-1'></a>创建Goroutine

创建一个goroutine通过Go 语言的 `go` 关键字，编译器会通过 [`cmd/compile/internal/gc.state.stmt`](https://draveness.me/golang/tree/cmd/compile/internal/gc.state.stmt) 和 [`cmd/compile/internal/gc.state.call`](https://draveness.me/golang/tree/cmd/compile/internal/gc.state.call) 两个方法将该关键字转换成 [`runtime.newproc`](https://draveness.me/golang/tree/runtime.newproc) 函数调用。

[`runtime.newproc`](https://draveness.me/golang/tree/runtime.newproc) 的入参是参数大小和表示函数的指针 `funcval`，它会获取 Goroutine 以及调用方的程序计数器，然后调用 [`runtime.newproc1`](https://draveness.me/golang/tree/runtime.newproc1) 函数获取新的 Goroutine 结构体、将其加入处理器的运行队列并在满足条件时调用 [`runtime.wakep`](https://draveness.me/golang/tree/runtime.wakep) 唤醒新的处理执行 Goroutine：

```go
func newproc(fn *funcval) {
	// 获取当前g的一个指针，编译器会重写这部分的函数来直接调用指令去从TLS中去获取
	gp := getg()
	// 创建该g的语句statement的地址
	pc := getcallerpc()
	systemstack(func() {
		newg := newproc1(fn, gp, pc)

		_p_ := getg().m.p.ptr()
		// runqput尝试将当前g放置到本地queue中，如果当前queue的next为false，那么放置到尾部
		// 如果next为true，那么放置到_p_.runnext槽位。如果runq满了，那么就会放置到全局queue中
		runqput(_p_, newg, true)

		// 如果主M已经启动
		if mainStarted {
			// wakep 会检测，如果有g处于runnable状态，那么尝试唤醒更多的p来执行g
			wakep()
		}
	})
}
```

[`runtime.newproc1`](https://draveness.me/golang/tree/runtime.newproc1) 会根据传入参数初始化一个 `g` 结构体，我们可以将该函数分成以下几个部分介绍它的实现：

1. 获取或者创建新的 Goroutine 结构体；
2. 将传入的参数移到 Goroutine 的栈上；
3. 更新 Goroutine 调度相关的属性；

###  3.2. <a name='Goroutine-1'></a>Goroutine 结构体的创建过程

```go
func newproc1(fn *funcval, argp unsafe.Pointer, narg int32, callergp *g, callerpc uintptr) *g {
	_g_ := getg()
	siz := narg
	siz = (siz + 7) &^ 7

	_p_ := _g_.m.p.ptr()
    // 从gFree列表中去获取空闲的goroutine，如果本地列表为空，从全局列表中去找一个
	newg := gfget(_p_)
	if newg == nil {
        // 如果获取不到，就调用malg来创建一个满足最小栈空间的g的结构体
		newg = malg(_StackMin)
	}
	...
```

上述代码会先从处理器的 `gFree` 列表中查找空闲的 Goroutine，如果不存在空闲的 Goroutine，会通过 `runtime.malg`创建一个栈大小足够的新结构体。

创建完g的结构体之后，`casgstatus`会设置刚创建的g的状态从`_Gidle`到`_Gdead`来防止GC扫描器扫描到这个未初始化完成的栈空间。

接下来，我们会调用 `runtime.memmove`将 `fn` 函数的所有参数拷贝到栈上，`argp` 和 `narg` 分别是参数的内存空间和大小，我们在该方法中会将参数对应的内存空间整块拷贝到栈上。

拷贝了栈上的参数之后，`runtime.newproc1` 会设置新的 Goroutine 结构体的参数，包括栈指针、程序计数器并更新其状态到 `_Grunnable` 并返回

###  3.3. <a name='Goroutine-1'></a>初始化Goroutine结构体

`runtime.gfget`通过两种不同的方式获取新的 `runtime.g`：

1. 从 Goroutine 所在处理器的 `gFree` 列表或者调度器的 `sched.gFree` 列表中获取 `runtime.g`；
2. 调用 `runtime.malg`生成一个新的 `runtime.g` 并将结构体追加到全局的 Goroutine 列表 `allgs` 中。

```go
func newproc1(fn *funcval, callergp *g, callerpc uintptr) *g {
	...

	_p_ := _g_.m.p.ptr()
	newg := gfget(_p_)
	if newg == nil {
		newg = malg(_StackMin)
		// 将创建的g的状态从_Gidle设置为_Gdead, 如果当前g的原子状态处于Gscan，那么该方法会循环直到Gscan结束
		casgstatus(newg, _Gidle, _Gdead)
		// 将正在创建的g的状态对外发布为Gdead，那么GC扫描器就无法扫描到当前未初始化完成的栈
		allgadd(newg) // publishes with a g->status of Gdead so GC scanner doesn't look at uninitialized stack.
	}
```

###  3.4. <a name='Goroutine-1'></a>设置Goroutine调度信息

```go
    // 把 newg.sched 结构体成员的所有成员设置为 0
    memclrNoHeapPointers(unsafe.Pointer(&newg.sched),unsafe.Sizeof(newg.sched))
    // 设置 newg 的 sched 成员，调度器需要依靠这些字段才能把 goroutine 调度到 CPU 上运行
    newg.sched.sp = sp
    newg.stktopsp = sp
    // newg.sched.pc 表示当 newg 被调度起来运行时从这个地址开始执行指令
    newg.sched.pc = funcPC(goexit)+ sys.PCQuantum// +PCQuantum so that previous instruction is in same function
    newg.sched.g = guintptr(unsafe.Pointer(newg))
    gostartcallfn(&newg.sched, fn)
    newg.gopc = callerpc
    // 设置 newg 的 startpc 为 fn.fn，该成员主要用于函数调用栈的 traceback 和栈收缩
    // newg 真正从哪里开始执行并不依赖于这个成员，而是 sched.pc
    newg.startpc = fn.fn
    if _g_.m.curg !=nil{
        newg.labels = _g_.m.curg.labels
    }
```

上面调度信息的设置只是设置了非常基本的`sp`, `pc`, `g`等信息，要真正完成Goroutine的初始化，还需要经过`runtime.gostartcallfn`和`runtime.gostartcall`两个函数。

```go
// adjust Gobuf as if it executed a call to fn
// and then stopped before the first instruction in fn.
func gostartcallfn(gobuf *gobuf, fv *funcval) {
	var fn unsafe.Pointer
	if fv != nil {
        // fn是gorountine的入口地址，初始化的时候对应的是runtime.main
		fn = unsafe.Pointer(fv.fn)
	} else {
		fn = unsafe.Pointer(abi.FuncPCABIInternal(nilfunc))
	}
	gostartcall(gobuf, fn, unsafe.Pointer(fv))
}

func gostartcall(buf *gobuf, fn, ctxt unsafe.Pointer) {
	// sp对应于newg的栈顶，目前newg栈上只有fn函数的参数，sp指向fn的第一个参数
    sp := buf.sp
    // 为返回地址预留空间
	sp -= goarch.PtrSize
    // 这里填的是newproc1函数里设置的goexit函数的第二条指令
    // 伪装成fn是被goexit函数调用的，使得fn执行完成之后返回到goexit继续执行一些清理操作
	*(*uintptr)(unsafe.Pointer(sp)) = buf.pc
    // 重新设置buf.sp
	buf.sp = sp
    // 当goroutine被调度起来执行的时候，会从这里的pc开始执行，初始化时就是runtime.main
	buf.pc = uintptr(fn)
	buf.ctxt = ctxt
}
```

**调度信息的 `sp` 中存储了 `runtime.goexit` 函数的程序计数器，而 `pc` 中存储了传入函数的程序计数器。因为 `pc` 寄存器的作用就是存储程序接下来运行的位置，所以 `pc` 的使用比较好理解。**

`buf`的作用就是一个**搬运工** 。当调度器调度到此goroutine时候，就取出`buf.sp`和`buf.pc`，恢复CPU的寄存器，重新构造goroutine的运行环境。

而`goexit`也将自己放置到`newg`的栈顶，每次goroutine执行完毕之后，都会执行到`goexit`从而完成清理工作。

