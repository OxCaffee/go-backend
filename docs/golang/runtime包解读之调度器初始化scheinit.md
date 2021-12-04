# runtime包源码解读——调度器初始化schedinit

<!-- vscode-markdown-toc -->
* 1. [前言](#)
* 2. [调度器初始化流程](#-1)
	* 2.1. [World停止](#World)
	* 2.2. [初始化调度器锁及优先级](#-1)
	* 2.3. [设置工作线程M的最大值](#M)
	* 2.4. [各项初始化](#-1)
		* 2.4.1. [堆栈初始化stackinit](#stackinit)
		* 2.4.2. [虚拟内存初始化mallocinit](#mallocinit)
	* 2.5. [设置最大逻辑处理器数](#-1)
	* 2.6. [垃圾收集GC初始化gcinit](#GCgcinit)
	* 2.7. [World开始运转](#World-1)
* 3. [总结](#-1)

<!-- vscode-markdown-toc-config
	numbering=true
	autoSave=true
	/vscode-markdown-toc-config -->
<!-- /vscode-markdown-toc -->

##  1. <a name=''></a>前言

前面的文章分析了golang程序的启动流程，在这里再次贴一下流程图:

<div align=center><img src="/assets/sched2.png"/></div>

这篇文章我们主要探讨一下`runtime.schedinit`调度器初始化。调度器初始化的源码位于`src/runtime/proc.go`文件内的`schedinit`方法。

##  2. <a name='-1'></a>调度器初始化流程

###  2.1. <a name='World'></a>World停止

```go
worldStopped()
```

调度器还没初始化完全，整个go world不能开始运行。

###  2.2. <a name='-1'></a>初始化调度器锁及优先级

```go
func schedinit() {
	lockInit(&sched.lock, lockRankSched)
	lockInit(&sched.sysmonlock, lockRankSysmon)
	lockInit(&sched.deferlock, lockRankDefer)
	lockInit(&sched.sudoglock, lockRankSudog)
	lockInit(&deadlock, lockRankDeadlock)
	lockInit(&paniclk, lockRankPanic)
	lockInit(&allglock, lockRankAllg)
	lockInit(&allpLock, lockRankAllp)
	lockInit(&reflectOffs.lock, lockRankReflectOffs)
	lockInit(&finlock, lockRankFin)
	lockInit(&trace.bufLock, lockRankTraceBuf)
	lockInit(&trace.stringsLock, lockRankTraceStrings)
	lockInit(&trace.lock, lockRankTrace)
	lockInit(&cpuprof.lock, lockRankCpuprof)
	lockInit(&trace.stackTab.lock, lockRankTraceStackTab)
	lockInit(&memstats.heapStats.noPLock, lockRankLeafRank)
    ...
}
```

###  2.3. <a name='M'></a>设置工作线程M的最大值

```go
func schedinit() {
    ...
    sched.maxmcount = 10000
    ...
}
```

**处于调度器和计算机性能的考虑，工作线程(内核线程)M的个数都不会超过10000**

###  2.4. <a name='-1'></a>各项初始化

```go
func schedinit() {
    ...
    stackinit()
	mallocinit()
	cpuinit()      // must run before alginit
	alginit()      // maps, hash, fastrand must not be used before this call
	fastrandinit() // must run before mcommoninit
	mcommoninit(_g_.m, -1)
	modulesinit()   // provides activeModules
	typelinksinit() // uses maps, activeModules
	itabsinit()     // uses activeModules
	stkobjinit()    // must run before GC starts
}
```

####  2.4.1. <a name='stackinit'></a>堆栈初始化stackinit

```go
func stackinit() {
	if _StackCacheSize&_PageMask != 0 {
		throw("cache size must be a multiple of page size")
	}
	for i := range stackpool {
		stackpool[i].item.span.init()
		lockInit(&stackpool[i].item.mu, lockRankStackpool)
	}
	for i := range stackLarge.free {
		stackLarge.free[i].init()
		lockInit(&stackLarge.lock, lockRankStackLarge)
	}
}
```

上面的代码一共初始化了两个栈相关的空间：

* **栈池(stack pool)** ：一个数组，每个数组保存的是包含相同大小栈的链表
* **GC用到的空闲链表(free)** ：也存储了一个栈的链表，但是这些栈都是在GC时被加入的，并且在回收的时候被清空。并且只有2KB, 4KB, 8KB以及16KB的栈才会被缓存，更大的栈会被直接分配。**有关内存分配的文章，golang官方有详细的源码注释，可以了解一下[内存分配官方注释](https://github.com/golang/go/blob/go1.5.1/src/runtime/malloc.go#L5)**

####  2.4.2. <a name='mallocinit'></a>虚拟内存初始化mallocinit

_虚拟内存的源码阅读需要先对Golang内存模型有个大概了解，所以这部分先跳过，后续再补上_

###  2.5. <a name='-1'></a>设置最大逻辑处理器数

```go
procs := ncpu
	if n, ok := atoi32(gogetenv("GOMAXPROCS")); ok && n > 0 {
		procs = n
	}
```

###  2.6. <a name='GCgcinit'></a>垃圾收集GC初始化gcinit

```go
func gcinit() {
	if unsafe.Sizeof(workbuf{}) != _WorkbufSize {
		throw("size of Workbuf is suboptimal")
	}
	// No sweep on the first cycle.
	sweep.active.state.Store(sweepDrainedMask)

	// Initialize GC pacer state.
	// Use the environment variable GOGC for the initial gcPercent value.
	gcController.init(readGOGC())

	work.startSema = 1
	work.markDoneSema = 1
	lockInit(&work.sweepWaiters.lock, lockRankSweepWaiters)
	lockInit(&work.assistQueue.lock, lockRankAssistQueue)
	lockInit(&work.wbufSpans.lock, lockRankWbufSpans)
}
```

###  2.7. <a name='World-1'></a>World开始运转

```go
// World is effectively started now, as P's can run.
	worldStarted()
```

##  3. <a name='-1'></a>总结

调度器初始化初始化了一系列的相关数据结构，包括，栈，堆，cpu，虚拟内存，缓存，表项，gc等等。**当schedinit执行完毕之后，整个golang世界就开始了运转。**
