# Golang runtime包源码解读——mgc(1)

<!-- vscode-markdown-toc -->
* 1. [不同阶段的GC行为](#GC)
	* 1.1. [Sweep Termination](#SweepTermination)
	* 1.2. [Mark Phase](#MarkPhase)
	* 1.3. [Mark Termination](#MarkTermination)
	* 1.4. [Sweep Phase](#SweepPhase)
* 2. [并发清扫(Concurrent Sweep)](#ConcurrentSweep)
* 3. [GC Rate](#GCRate)
* 4. [Oblets分割](#Oblets)

<!-- vscode-markdown-toc-config
	numbering=true
	autoSave=true
	/vscode-markdown-toc-config -->
<!-- /vscode-markdown-toc -->

##  1. <a name='GC'></a>不同阶段的GC行为

###  1.1. <a name='SweepTermination'></a>Sweep Termination

1. STW，这会让所有的P都到达安全点`safe point`
2. 清除所有未被清除的`spans`，注意，未被清除的`spans`的存在只有一种可能：**GC循环被强制提前执行，导致还有`spans`未被清理**

###  1.2. <a name='MarkPhase'></a>Mark Phase

1. 设置`gcphase`变量从`_GCoff`到`_Gcmark` 
2. 开启所有的写屏障`write barrier`
3. 将根标记任务`root mark job`入队列(**注意，在所有的P开启写屏障之前，不会有任何的对象被扫描** )
4. `Start the world` 。从这个时间点开始，GC工作将由调度器指定的`mark workers`和内存分配器的协助完成。写屏障将会**着色(shading)所有对重写指针(overwritten pointer)和新指针(new pointer)** 。新分配的指针对象会被立即标注为黑色。

```markdown
疑问: 何为overwritten指针？这个疑问可能要等mbarrier.go源码读完之后才知道。
```

5. 根标记任务将会被执行(`root marking jobs`) 。执行的操作包括扫描所有堆栈，着色所有全局变量，以及着色任何堆指针、堆外运行时数据结构。扫描堆栈会停止一个 goroutine，遮蔽在其堆栈上找到的任何指针，然后恢复 goroutine。

6. 取出所有的灰色对象，将所有的灰色对象标注为黑色对象，并将与该灰色对象“连接”的对象标记为灰色(这意味着又有新的指针被加入队列)
7. GC使用了分布式标记终止算法来检测没有其他标记任务或者灰色对象(详情请见`gcMarkDone`) 。至此，GC任务将会过度到`mark termination`阶段。

###  1.3. <a name='MarkTermination'></a>Mark Termination

1. STW
2. 将`gcphase`变量设置为`_GCmarktermination` ，禁用所有的`gc workers`和`gc assists`

###  1.4. <a name='SweepPhase'></a>Sweep Phase

1. 将`gcphase`设置为`_GCoff` ，设置sweep状态并且禁用所有的写屏障
2. `Start the world` 。从这个时间点开始，新分配的对象为白色。**必要时可以分配待清扫的spans去使用**

```markdown
go源码中原文是这么说的：
Start the world. From this point on, newly allocated objects are white, and allocating sweeps spans before use if necessary
```

3. GC 在后台执行并发清除的同时并响应分配。

##  2. <a name='ConcurrentSweep'></a>并发清扫(Concurrent Sweep)

在清扫阶段(`sweep phase`)，所有的程序正常运行，堆内存中的`span`会\一个一个地被回收，并且是采用懒回收地方式(当一个goroutine需要另外一个span的时候才会触发清扫)，清扫任务交由一个后台goroutine执行(`background`)，在STW `mark termination`阶段，所有的`spans`都会被标记为`needs sweeping` 。

为了避免请求过多的OS内存，当一个goroutine需要另外一个`span`的时候，它会首先执行回收操作`reclaim memory by sweeping`。

* 当一个goroutine需要一个小对象(`small-size-object`)，会优先回收小对象大小的`span`
* 当一个goroutine需要一个大对象(`large-size-object`)，会优先回收大对象大小的`span`

**在一种情况下这可能还不够：如果一个 goroutine 将两个不相邻的单页跨度清除并释放到堆中，它将分配一个新的两页跨度，但仍然可以有其他一页未扫描跨度，这种情况下仍然可以被合并成一个两页的span。**

##  3. <a name='GCRate'></a>GC Rate

`Next GC`发生在分配了与已使用内存成正比的额外内存之后。这个比例由`GOGC`环境变量控制，例如`GOGC=100`，当使用了4M内存，那么下一次GC的内存将会是8M，这个比例是线性的。

##  4. <a name='Oblets'></a>Oblets分割

为了避免扫描大对象造成的gc停顿，GC会将超过`maxObletsBytes`的大对象分割为`maxObletsBytes`大小的对象。分割后的对象会被当成新的对象处理后面的GC。
