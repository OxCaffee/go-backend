# Go语言并发之同步原语和锁

<!-- vscode-markdown-toc -->
* 1. [前言](#)
* 2. [Go语言中的同步原语](#Go)
* 3. [Mutex](#Mutex)
	* 3.1. [状态state](#state)
	* 3.2. [Mutex的正常模式和饥饿模式](#Mutex-1)
	* 3.3. [加锁和解锁](#-1)
		* 3.3.1. [互斥锁如何判断当前Goroutine能否进入自旋](#Goroutine)
	* 3.4. [总结一下互斥锁的加锁和解锁流程](#-1)
* 4. [RWMutex](#RWMutex)
	* 4.1. [ 写锁](#-1)
	* 4.2. [读锁](#-1)
* 5. [WaitGroup](#WaitGroup)
* 6. [Once](#Once)
* 7. [Cond(不常用，一般用channel代替)](#Condchannel)
* 8. [ErrGroup](#ErrGroup)
	* 8.1. [总结](#-1)
* 9. [Semaphore](#Semaphore)
	* 9.1. [获取信号量](#-1)
	* 9.2. [释放信号量](#-1)
	* 9.3. [总结](#-1)
* 10. [SingleFlight](#SingleFlight)

<!-- vscode-markdown-toc-config
	numbering=true
	autoSave=true
	/vscode-markdown-toc-config -->
<!-- /vscode-markdown-toc -->

##  1. <a name=''></a>前言

本文件的内容主要参考了[同步原语和锁](https://draveness.me/golang/docs/part3-runtime/ch06-concurrency/golang-sync-primitives/#cond)博客

##  2. <a name='Go'></a>Go语言中的同步原语

* `sync.Mutex`
* `sync.RWMutex`
* `sync.WaitGroup`
* `sync.Once`
* `sync.Cond`

以及一些拓展原语：

* `golang/sync/errgroup.Group`
* `golang/sync/semaphore.Weighted`
* `golang/sync/singleflight.Group`

这些基本原语提供了较为基础的同步功能，但是它们是一种相对原始的同步机制，在多数情况下，我们都应该使用抽象层级更高的 Channel 实现同步。

##  3. <a name='Mutex'></a>Mutex

Go 语言的 [`sync.Mutex`](https://draveness.me/golang/tree/sync.Mutex) 由两个字段 `state` 和 `sema` 组成。其中 `state` 表示当前互斥锁的状态，而 `sema` 是用于控制锁状态的信号量。

```go
type Mutex struct {
	state int32
	sema  uint32
}
```

上述两个加起来只占 8 字节空间的结构体表示了 Go 语言中的互斥锁。

###  3.1. <a name='state'></a>状态state

互斥锁的状态比较复杂，如下图所示，最低三位分别表示 `mutexLocked`、`mutexWoken` 和 `mutexStarving`，剩下的位置用来表示当前有多少个 Goroutine 在等待互斥锁的释放。

互斥锁的状态state是一个32位的int值，占8个字节，**在默认情况下，所有的状态位都是0** 。`int32`中不同的位分别表示了不同的状态：

* `mutextLocked` ：表示互斥锁正处于锁定状态
* `mutexWoken` ：表示从正常模式被唤醒
* `mutexStarving` ：当前的互斥锁进入饥饿状态
* `waitersCount` ：当前的互斥锁上等待的Goroutine数目

###  3.2. <a name='Mutex-1'></a>Mutex的正常模式和饥饿模式

[`sync.Mutex`](https://draveness.me/golang/tree/sync.Mutex) 有两种模式 — 正常模式和饥饿模式。我们需要在这里先了解正常模式和饥饿模式都是什么以及它们有什么样的关系。

在正常模式下，锁的等待者会按照先进先出的顺序获取锁。但是刚被唤起的 Goroutine 与新创建的 Goroutine 竞争时，大概率会获取不到锁，为了减少这种情况的出现，**一旦 Goroutine 超过 1ms 没有获取到锁，它就会将当前互斥锁切换饥饿模式，防止部分 Goroutine 被『饿死』**。

在饥饿模式中，互斥锁会直接交给等待队列最前面的 Goroutine。新的 Goroutine 在该状态下不能获取锁、也不会进入自旋状态，它们只会在队列的末尾等待。如果一个 Goroutine 获得了互斥锁并且它在队列的末尾或者它等待的时间少于 1ms，那么当前的互斥锁就会切换回正常模式。简而言之，**新创建的锁在饥饿模式下不能竞争锁** 。

###  3.3. <a name='-1'></a>加锁和解锁

 [`sync.Mutex.Lock`](https://draveness.me/golang/tree/sync.Mutex.Lock) 和 [`sync.Mutex.Unlock`](https://draveness.me/golang/tree/sync.Mutex.Unlock) 方法分别对应着加锁和解锁。

互斥锁的加锁是靠 [`sync.Mutex.Lock`](https://draveness.me/golang/tree/sync.Mutex.Lock) 完成的，最新的 Go 语言源代码中已经将 [`sync.Mutex.Lock`](https://draveness.me/golang/tree/sync.Mutex.Lock) 方法进行了简化，方法的主干只保留最常见、简单的情况 — 当锁的状态是 0 时，将 `mutexLocked` 位置成 1：

```go
func (m *Mutex) Lock() {
    // 锁状态是0，去竞争锁
	if atomic.CompareAndSwapInt32(&m.state, 0, mutexLocked) {
		return
	}
    // 锁的状态是1，自旋获取锁
	m.lockSlow()
}
```

如果互斥锁的状态不是 0 时就会调用 [`sync.Mutex.lockSlow`](https://draveness.me/golang/tree/sync.Mutex.lockSlow) 尝试通过自旋（Spinnig）等方式等待锁的释放，该方法的主体是一个非常大 for 循环，这里将它分成几个部分介绍获取锁的过程：

1. 判断当前 Goroutine 能否进入自旋；
2. 通过自旋等待互斥锁的释放；
3. 计算互斥锁的最新状态；
4. 更新互斥锁的状态并获取锁；

####  3.3.1. <a name='Goroutine'></a>互斥锁如何判断当前Goroutine能否进入自旋

```go
func (m *Mutex) lockSlow() {
	var waitStartTime int64
	starving := false
	awoke := false
	iter := 0
	old := m.state
	for {
		if old&(mutexLocked|mutexStarving) == mutexLocked && runtime_canSpin(iter) {
			if !awoke && old&mutexWoken == 0 && old>>mutexWaiterShift != 0 &&
				atomic.CompareAndSwapInt32(&m.state, old, old|mutexWoken) {
				awoke = true
			}
			runtime_doSpin()
			iter++
			old = m.state
			continue
		}
```

自旋是一种多线程同步机制，当前的进程在进入自旋的过程中会一直保持 CPU 的占用，持续检查某个条件是否为真。在多核的 CPU 上，自旋可以避免 Goroutine 的切换，使用恰当会对性能带来很大的增益，但是使用的不恰当就会拖慢整个程序，所以 Goroutine 进入自旋的条件非常苛刻：

1. 互斥锁只有在普通模式才能进入自旋；
2. `runtime.sync_runtime_canSpin`需要返回`true` :
   1. 运行在多 CPU 的机器上；
   2. 当前 Goroutine 为了获取该锁进入自旋的次数小于四次；
   3. 当前机器上至少存在一个正在运行的处理器 P 并且处理的运行队列为空；

一旦当前 Goroutine 能够进入自旋就会调用[`runtime.sync_runtime_doSpin`](https://draveness.me/golang/tree/runtime.sync_runtime_doSpin) 和 [`runtime.procyield`](https://draveness.me/golang/tree/runtime.procyield) 并执行 30 次的 `PAUSE` 指令，该指令只会占用 CPU 并消耗 CPU 时间：

```go
func sync_runtime_doSpin() {
	procyield(active_spin_cnt)
}

TEXT runtime·procyield(SB),NOSPLIT,$0-0
	MOVL	cycles+0(FP), AX
again:
	PAUSE
	SUBL	$1, AX
	JNZ	again
	RET
```

处理了自旋相关的特殊逻辑之后，互斥锁会根据上下文计算当前互斥锁最新的状态。几个不同的条件分别会更新 `state` 字段中存储的不同信息 — `mutexLocked`、`mutexStarving`、`mutexWoken` 和 `mutexWaiterShift`：

```go
		new := old
		if old&mutexStarving == 0 {
			new |= mutexLocked
		}
		if old&(mutexLocked|mutexStarving) != 0 {
			new += 1 << mutexWaiterShift
		}
		if starving && old&mutexLocked != 0 {
			new |= mutexStarving
		}
		if awoke {
			new &^= mutexWoken
		}
```

计算了新的互斥锁状态之后，会使用 CAS 函数 [`sync/atomic.CompareAndSwapInt32`](https://draveness.me/golang/tree/sync/atomic.CompareAndSwapInt32) 更新状态：

```go
		if atomic.CompareAndSwapInt32(&m.state, old, new) {
			if old&(mutexLocked|mutexStarving) == 0 {
				break // 通过 CAS 函数获取了锁
			}
			...
			runtime_SemacquireMutex(&m.sema, queueLifo, 1)
			starving = starving || runtime_nanotime()-waitStartTime > starvationThresholdNs
			old = m.state
			if old&mutexStarving != 0 {
				delta := int32(mutexLocked - 1<<mutexWaiterShift)
				if !starving || old>>mutexWaiterShift == 1 {
					delta -= mutexStarving
				}
				atomic.AddInt32(&m.state, delta)
				break
			}
			awoke = true
			iter = 0
		} else {
			old = m.state
		}
	}
}
```

如果没有通过 CAS 获得锁，会调用 [`runtime.sync_runtime_SemacquireMutex`](https://draveness.me/golang/tree/runtime.sync_runtime_SemacquireMutex) 通过信号量保证资源不会被两个 Goroutine 获取。[`runtime.sync_runtime_SemacquireMutex`](https://draveness.me/golang/tree/runtime.sync_runtime_SemacquireMutex) 会在方法中不断尝试获取锁并陷入休眠等待信号量的释放，一旦当前 Goroutine 可以获取信号量，它就会立刻返回，[`sync.Mutex.Lock`](https://draveness.me/golang/tree/sync.Mutex.Lock) 的剩余代码也会继续执行。

- 在正常模式下，这段代码会设置唤醒和饥饿标记、重置迭代次数并重新执行获取锁的循环；
- 在饥饿模式下，当前 Goroutine 会获得互斥锁，如果等待队列中只存在当前 Goroutine，互斥锁还会从饥饿模式中退出；

互斥锁的解锁过程 [`sync.Mutex.Unlock`](https://draveness.me/golang/tree/sync.Mutex.Unlock) 与加锁过程相比就很简单，该过程会先使用 [`sync/atomic.AddInt32`](https://draveness.me/golang/tree/sync/atomic.AddInt32) 函数快速解锁，这时会发生下面的两种情况：

- 如果该函数返回的新状态等于 0，当前 Goroutine 就成功解锁了互斥锁；
- 如果该函数返回的新状态不等于 0，这段代码会调用 [`sync.Mutex.unlockSlow`](https://draveness.me/golang/tree/sync.Mutex.unlockSlow) 开始慢速解锁：

```go
func (m *Mutex) Unlock() {
	new := atomic.AddInt32(&m.state, -mutexLocked)
	if new != 0 {
		m.unlockSlow(new)
	}
}
```

[`sync.Mutex.unlockSlow`](https://draveness.me/golang/tree/sync.Mutex.unlockSlow) 会先校验锁状态的合法性 — 如果当前互斥锁已经被解锁过了会直接抛出异常 “sync: unlock of unlocked mutex” 中止当前程序。

在正常情况下会根据当前互斥锁的状态，分别处理正常模式和饥饿模式下的互斥锁：

```go
func (m *Mutex) unlockSlow(new int32) {
	if (new+mutexLocked)&mutexLocked == 0 {
		throw("sync: unlock of unlocked mutex")
	}
	if new&mutexStarving == 0 { // 正常模式
		old := new
		for {
			if old>>mutexWaiterShift == 0 || old&(mutexLocked|mutexWoken|mutexStarving) != 0 {
				return
			}
			new = (old - 1<<mutexWaiterShift) | mutexWoken
			if atomic.CompareAndSwapInt32(&m.state, old, new) {
				runtime_Semrelease(&m.sema, false, 1)
				return
			}
			old = m.state
		}
	} else { // 饥饿模式
		runtime_Semrelease(&m.sema, true, 1)
	}
}
```

- 在正常模式下，上述代码会使用如下所示的处理过程：
  - 如果互斥锁不存在等待者或者互斥锁的 `mutexLocked`、`mutexStarving`、`mutexWoken` 状态不都为 0，那么当前方法可以直接返回，不需要唤醒其他等待者；
  - 如果互斥锁存在等待者，会通过 [`sync.runtime_Semrelease`](https://draveness.me/golang/tree/sync.runtime_Semrelease) 唤醒等待者并移交锁的所有权；
- 在饥饿模式下，上述代码会直接调用 [`sync.runtime_Semrelease`](https://draveness.me/golang/tree/sync.runtime_Semrelease) 将当前锁交给下一个正在尝试获取锁的等待者，等待者被唤醒后会得到锁，在这时互斥锁还不会退出饥饿状态；

###  3.4. <a name='-1'></a>总结一下互斥锁的加锁和解锁流程

**互斥锁的加锁过程比较复杂，它涉及自旋、信号量以及调度等概念：**

- 如果互斥锁处于初始化状态，会通过置位 `mutexLocked` 加锁；
- 如果互斥锁处于 `mutexLocked` 状态并且在普通模式下工作，会进入自旋，执行 30 次 `PAUSE` 指令消耗 CPU 时间等待锁的释放；
- 如果当前 Goroutine 等待锁的时间超过了 1ms，互斥锁就会切换到饥饿模式；
- 互斥锁在正常情况下会通过 [`runtime.sync_runtime_SemacquireMutex`](https://draveness.me/golang/tree/runtime.sync_runtime_SemacquireMutex) 将尝试获取锁的 Goroutine 切换至休眠状态，等待锁的持有者唤醒；
- **如果当前 Goroutine 是互斥锁上的最后一个等待的协程或者等待的时间小于 1ms，那么它会将互斥锁切换回正常模式；**

**互斥锁的解锁过程与之相比就比较简单，其代码行数不多、逻辑清晰，也比较容易理解：**

- 当互斥锁已经被解锁时，调用 [`sync.Mutex.Unlock`](https://draveness.me/golang/tree/sync.Mutex.Unlock) 会直接抛出异常；
- 当互斥锁处于饥饿模式时，将锁的所有权交给队列中的下一个等待者，等待者会负责设置 `mutexLocked` 标志位；
- 当互斥锁处于普通模式时，如果没有 Goroutine 等待锁的释放或者已经有被唤醒的 Goroutine 获得了锁，会直接返回；在其他情况下会通过 [`sync.runtime_Semrelease`](https://draveness.me/golang/tree/sync.runtime_Semrelease) 唤醒对应的 Goroutine；

**四种会发生错误的操作一定要避免**

- 解锁已经解锁的锁，会Panic
- 复制这个锁，因为底层来看都是有状态的，复制以后就不是初始值了，肯定会出错
- mutex不支持重入，意思就是某个持有的goroutine可劲再调用锁，go不支持，因为go的锁不记录goroutine的信息。

##  4. <a name='RWMutex'></a>RWMutex

读写互斥锁 [`sync.RWMutex`](https://draveness.me/golang/tree/sync.RWMutex) 是细粒度的互斥锁，它不限制资源的并发读，但是读写、写写操作无法并行执行。

[`sync.RWMutex`](https://draveness.me/golang/tree/sync.RWMutex) 中总共包含以下 5 个字段：

```go
type RWMutex struct {
	w           Mutex
	writerSem   uint32
	readerSem   uint32
	readerCount int32
	readerWait  int32
}
```

- `w` — 复用互斥锁提供的能力；
- `writerSem` 和 `readerSem` — 分别用于写等待读和读等待写：
- `readerCount` 存储了当前正在执行的读操作数量；
- `readerWait` 表示当写操作被阻塞时等待的读操作个数；

我们会依次分析获取写锁和读锁的实现原理，其中：

- 写操作使用 [`sync.RWMutex.Lock`](https://draveness.me/golang/tree/sync.RWMutex.Lock) 和 [`sync.RWMutex.Unlock`](https://draveness.me/golang/tree/sync.RWMutex.Unlock) 方法；
- 读操作使用 [`sync.RWMutex.RLock`](https://draveness.me/golang/tree/sync.RWMutex.RLock) 和 [`sync.RWMutex.RUnlock`](https://draveness.me/golang/tree/sync.RWMutex.RUnlock) 方法；

###  4.1. <a name='-1'></a> 写锁

当资源的使用者想要获取写锁时，需要调用 [`sync.RWMutex.Lock`](https://draveness.me/golang/tree/sync.RWMutex.Lock) 方法：

```go
func (rw *RWMutex) Lock() {
	rw.w.Lock()
	r := atomic.AddInt32(&rw.readerCount, -rwmutexMaxReaders) + rwmutexMaxReaders
	if r != 0 && atomic.AddInt32(&rw.readerWait, r) != 0 {
		runtime_SemacquireMutex(&rw.writerSem, false, 0)
	}
}
```

1. 调用`sync.Mutex` 结构体的`sync.Mutex.Lock`阻塞后续的写操作，因为互斥锁已经被获取，因此其他Goroutine在获取写锁时会进入自旋或者睡眠
2. 调用`sync/atomic.AddInt32`函数阻塞后续的读操作
3. 如果仍然有其他Goroutine持有互斥锁的读锁，该Gouroutine会调用`runtime.sync_runtime_SemacquireMutex`进入休眠状态等待所有读锁所有者执行结束之后释放`writeSem`信号量将当前协程唤醒

写锁的释放会调用 [`sync.RWMutex.Unlock`](https://draveness.me/golang/tree/sync.RWMutex.Unlock)：

```go
func (rw *RWMutex) Unlock() {
	r := atomic.AddInt32(&rw.readerCount, rwmutexMaxReaders)
	if r >= rwmutexMaxReaders {
		throw("sync: Unlock of unlocked RWMutex")
	}
	for i := 0; i < int(r); i++ {
		runtime_Semrelease(&rw.readerSem, false, 0)
	}
	rw.w.Unlock()
}
```

与加锁的过程正好相反，写锁的释放分以下几个执行：

1. 调用 [`sync/atomic.AddInt32`](https://draveness.me/golang/tree/sync/atomic.AddInt32) 函数将 `readerCount` 变回正数，释放读锁；
2. 通过 for 循环释放所有因为获取读锁而陷入等待的 Goroutine：
3. 调用 [`sync.Mutex.Unlock`](https://draveness.me/golang/tree/sync.Mutex.Unlock) 释放写锁；

获取写锁时会先阻塞写锁的获取，后阻塞读锁的获取，这种策略能够保证读操作不会被连续的写操作『饿死』。

###  4.2. <a name='-1'></a>读锁

读锁的加锁方法 [`sync.RWMutex.RLock`](https://draveness.me/golang/tree/sync.RWMutex.RLock) 很简单，该方法会通过 [`sync/atomic.AddInt32`](https://draveness.me/golang/tree/sync/atomic.AddInt32) 将 `readerCount` 加一：

```go
func (rw *RWMutex) RLock() {
	if atomic.AddInt32(&rw.readerCount, 1) < 0 {
		runtime_SemacquireMutex(&rw.readerSem, false, 0)
	}
}
```

1. 如果该方法返回负数 — 其他 Goroutine 获得了写锁，当前 Goroutine 就会调用 [`runtime.sync_runtime_SemacquireMutex`](https://draveness.me/golang/tree/runtime.sync_runtime_SemacquireMutex) 陷入休眠等待锁的释放；
2. 如果该方法的结果为非负数,，即没有 Goroutine 获得写锁，当前方法会成功返回；

当 Goroutine 想要释放读锁时，会调用如下所示的 [`sync.RWMutex.RUnlock`](https://draveness.me/golang/tree/sync.RWMutex.RUnlock) 方法：

```go
func (rw *RWMutex) RUnlock() {
	if r := atomic.AddInt32(&rw.readerCount, -1); r < 0 {
		rw.rUnlockSlow(r)
	}
}
```

该方法会先减少正在读资源的 `readerCount` 整数，根据 [`sync/atomic.AddInt32`](https://draveness.me/golang/tree/sync/atomic.AddInt32) 的返回值不同会分别进行处理：

- 如果返回值大于等于零 — 读锁直接解锁成功；
- 如果返回值小于零 — 有一个正在执行的写操作，在这时会调用[`sync.RWMutex.rUnlockSlow`](https://draveness.me/golang/tree/sync.RWMutex.rUnlockSlow) 方法；

```go
func (rw *RWMutex) rUnlockSlow(r int32) {
	if r+1 == 0 || r+1 == -rwmutexMaxReaders {
		throw("sync: RUnlock of unlocked RWMutex")
	}
	if atomic.AddInt32(&rw.readerWait, -1) == 0 {
		runtime_Semrelease(&rw.writerSem, false, 1)
	}
}
```

[`sync.RWMutex.rUnlockSlow`](https://draveness.me/golang/tree/sync.RWMutex.rUnlockSlow) 会减少获取锁的写操作等待的读操作数 `readerWait` 并在所有读操作都被释放之后触发写操作的信号量 `writerSem`，该信号量被触发时，调度器就会唤醒尝试获取写锁的 Goroutine。

##  5. <a name='WaitGroup'></a>WaitGroup

[`sync.WaitGroup`](https://draveness.me/golang/tree/sync.WaitGroup) 结构体中只包含两个成员变量：

```go
type WaitGroup struct {
	noCopy noCopy
	state1 [3]uint32
}
```

- `noCopy` — 保证 [`sync.WaitGroup`](https://draveness.me/golang/tree/sync.WaitGroup) 不会被开发者通过再赋值的方式拷贝；
- `state1` — 存储着状态和信号量；

[`sync.noCopy`](https://draveness.me/golang/tree/sync.noCopy) 是一个特殊的私有结构体，[`tools/go/analysis/passes/copylock`](https://pkg.go.dev/golang.org/x/tools/go/analysis/passes/copylock) 包中的分析器会在编译期间检查被拷贝的变量中是否包含 [`sync.noCopy`](https://draveness.me/golang/tree/sync.noCopy) 或者实现了 `Lock` 和 `Unlock` 方法，如果包含该结构体或者实现了对应的方法就会报出以下错误：

```go
func main() {
	wg := sync.WaitGroup{}
	yawg := wg
	fmt.Println(wg, yawg)
}

$ go vet proc.go
./prog.go:10:10: assignment copies lock value to yawg: sync.WaitGroup
./prog.go:11:14: call of fmt.Println copies lock value: sync.WaitGroup
./prog.go:11:18: call of fmt.Println copies lock value: sync.WaitGroup
```

这段代码会因为变量赋值或者调用函数时发生值拷贝导致分析器报错。

除了 [`sync.noCopy`](https://draveness.me/golang/tree/sync.noCopy) 之外，[`sync.WaitGroup`](https://draveness.me/golang/tree/sync.WaitGroup)` 结构体中还包含一个总共占用 12 字节的数组，这个数组会存储当前结构体的状态，在 64 位与 32 位的机器上表现也非常不同。

[`sync.WaitGroup`](https://draveness.me/golang/tree/sync.WaitGroup) 对外暴露了三个方法 — [`sync.WaitGroup.Add`](https://draveness.me/golang/tree/sync.WaitGroup.Add)、[`sync.WaitGroup.Wait`](https://draveness.me/golang/tree/sync.WaitGroup.Wait) 和 [`sync.WaitGroup.Done`](https://draveness.me/golang/tree/sync.WaitGroup.Done)。

因为其中的 [`sync.WaitGroup.Done`](https://draveness.me/golang/tree/sync.WaitGroup.Done) 只是向 [`sync.WaitGroup.Add`](https://draveness.me/golang/tree/sync.WaitGroup.Add) 方法传入了 -1，所以我们重点分析另外两个方法，即 [`sync.WaitGroup.Add`](https://draveness.me/golang/tree/sync.WaitGroup.Add) 和 [`sync.WaitGroup.Wait`](https://draveness.me/golang/tree/sync.WaitGroup.Wait)：

```go
func (wg *WaitGroup) Add(delta int) {
	statep, semap := wg.state()
	state := atomic.AddUint64(statep, uint64(delta)<<32)
	v := int32(state >> 32)
	w := uint32(state)
	if v < 0 {
		panic("sync: negative WaitGroup counter")
	}
	if v > 0 || w == 0 {
		return
	}
	*statep = 0
	for ; w != 0; w-- {
		runtime_Semrelease(semap, false, 0)
	}
}
```

[`sync.WaitGroup.Add`](https://draveness.me/golang/tree/sync.WaitGroup.Add) 可以更新 [`sync.WaitGroup`](https://draveness.me/golang/tree/sync.WaitGroup) 中的计数器 `counter`。虽然 [`sync.WaitGroup.Add`](https://draveness.me/golang/tree/sync.WaitGroup.Add) 方法传入的参数可以为负数，但是计数器只能是非负数，一旦出现负数就会发生程序崩溃。当调用计数器归零，即所有任务都执行完成时，才会通过 [`sync.runtime_Semrelease`](https://draveness.me/golang/tree/sync.runtime_Semrelease) 唤醒处于等待状态的 Goroutine。

[`sync.WaitGroup`](https://draveness.me/golang/tree/sync.WaitGroup) 的另一个方法 [`sync.WaitGroup.Wait`](https://draveness.me/golang/tree/sync.WaitGroup.Wait) 会在计数器大于 0 并且不存在等待的 Goroutine 时，调用 [`runtime.sync_runtime_Semacquire`](https://draveness.me/golang/tree/runtime.sync_runtime_Semacquire) 陷入睡眠。

```go
func (wg *WaitGroup) Wait() {
	statep, semap := wg.state()
	for {
		state := atomic.LoadUint64(statep)
		v := int32(state >> 32)
		if v == 0 {
			return
		}
		if atomic.CompareAndSwapUint64(statep, state, state+1) {
			runtime_Semacquire(semap)
			if +statep != 0 {
				panic("sync: WaitGroup is reused before previous Wait has returned")
			}
			return
		}
	}
}
```

当 [`sync.WaitGroup`](https://draveness.me/golang/tree/sync.WaitGroup) 的计数器归零时，陷入睡眠状态的 Goroutine 会被唤醒，上述方法也会立刻返回。

##  6. <a name='Once'></a>Once

Go 语言标准库中 [`sync.Once`](https://draveness.me/golang/tree/sync.Once) 可以保证在 Go 程序运行期间的某段代码只会执行一次。在运行如下所示的代码时，我们会看到如下所示的运行结果：

```go
func main() {
    o := &sync.Once{}
    for i := 0; i < 10; i++ {
        o.Do(func() {
            fmt.Println("only once")
        })
    }
}

$ go run main.go
only once
```

每一个 [`sync.Once`](https://draveness.me/golang/tree/sync.Once) 结构体中都只包含一个用于标识代码块是否执行过的 `done` 以及一个互斥锁 [`sync.Mutex`](https://draveness.me/golang/tree/sync.Mutex)：

```go
type Once struct {
	done uint32
	m    Mutex
}
```

[`sync.Once.Do`](https://draveness.me/golang/tree/sync.Once.Do) 是 [`sync.Once`](https://draveness.me/golang/tree/sync.Once) 结构体对外唯一暴露的方法，该方法会接收一个入参为空的函数：

- 如果传入的函数已经执行过，会直接返回；
- 如果传入的函数没有执行过，会调用 [`sync.Once.doSlow`](https://draveness.me/golang/tree/sync.Once.doSlow) 执行传入的函数：

```go
func (o *Once) Do(f func()) {
	if atomic.LoadUint32(&o.done) == 0 {
		o.doSlow(f)
	}
}

func (o *Once) doSlow(f func()) {
	o.m.Lock()
	defer o.m.Unlock()
	if o.done == 0 {
		defer atomic.StoreUint32(&o.done, 1)
		f()
	}
}
```

1. 为当前 Goroutine 获取互斥锁；
2. 执行传入的无入参函数；
3. 运行延迟函数调用，将成员变量 `done` 更新成 1；

[`sync.Once`](https://draveness.me/golang/tree/sync.Once) 会通过成员变量 `done` 确保函数不会执行第二次。

##  7. <a name='Condchannel'></a>Cond(不常用，一般用channel代替)

##  8. <a name='ErrGroup'></a>ErrGroup

[`golang/sync/errgroup.Group`](https://draveness.me/golang/tree/golang/sync/errgroup.Group) 为我们在一组 Goroutine 中提供了同步、错误传播以及上下文取消的功能，我们可以使用如下所示的方式并行获取网页的数据：

```go
var g errgroup.Group
var urls = []string{
    "http://www.golang.org/",
    "http://www.google.com/",
    "http://www.somestupidname.com/",
}
for i := range urls {
    url := urls[i]
    g.Go(func() error {
        resp, err := http.Get(url)
        if err == nil {
            resp.Body.Close()
        }
        return err
    })
}
if err := g.Wait(); err == nil {
    fmt.Println("Successfully fetched all URLs.")
}
```

[`golang/sync/errgroup.Group.Go`](https://draveness.me/golang/tree/golang/sync/errgroup.Group.Go) 方法能够创建一个 Goroutine 并在其中执行传入的函数，而 [`golang/sync/errgroup.Group.Wait`](https://draveness.me/golang/tree/golang/sync/errgroup.Group.Wait) 会等待所有 Goroutine 全部返回，该方法的不同返回结果也有不同的含义：

- 如果返回错误 — 这一组 Goroutine 最少返回一个错误；
- 如果返回空值 — 所有 Goroutine 都成功执行；

[`golang/sync/errgroup.Group`](https://draveness.me/golang/tree/golang/sync/errgroup.Group) 结构体同时由三个比较重要的部分组成：

1. `cancel` — 创建 [`context.Context`](https://draveness.me/golang/tree/context.Context) 时返回的取消函数，用于在多个 Goroutine 之间同步取消信号；
2. `wg` — 用于等待一组 Goroutine 完成子任务的同步原语；
3. `errOnce` — 用于保证只接收一个子任务返回的错误；

```go
type Group struct {
	cancel func()

	wg sync.WaitGroup

	errOnce sync.Once
	err     error
}
```

这些字段共同组成了 [`golang/sync/errgroup.Group`](https://draveness.me/golang/tree/golang/sync/errgroup.Group) 结构体并为我们提供同步、错误传播以及上下文取消等功能。

我们能通过 [`golang/sync/errgroup.WithContext`](https://draveness.me/golang/tree/golang/sync/errgroup.WithContext) 构造器创建新的 [`golang/sync/errgroup.Group`](https://draveness.me/golang/tree/golang/sync/errgroup.Group) 结构体：

```go
func WithContext(ctx context.Context) (*Group, context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	return &Group{cancel: cancel}, ctx
}
```

运行新的并行子任务需要使用 [`golang/sync/errgroup.Group.Go`](https://draveness.me/golang/tree/golang/sync/errgroup.Group.Go) 方法，这个方法的执行过程如下：

1. 调用 [`sync.WaitGroup.Add`](https://draveness.me/golang/tree/sync.WaitGroup.Add) 增加待处理的任务；
2. 创建新的 Goroutine 并运行子任务；
3. 返回错误时及时调用 `cancel` 并对 `err` 赋值，只有最早返回的错误才会被上游感知到，后续的错误都会被舍弃：

```go
func (g *Group) Go(f func() error) {
	g.wg.Add(1)

	go func() {
		defer g.wg.Done()

		if err := f(); err != nil {
			g.errOnce.Do(func() {
				g.err = err
				if g.cancel != nil {
					g.cancel()
				}
			})
		}
	}()
}

func (g *Group) Wait() error {
	g.wg.Wait()
	if g.cancel != nil {
		g.cancel()
	}
	return g.err
}
```

另一个用于等待的 [`golang/sync/errgroup.Group.Wait`](https://draveness.me/golang/tree/golang/sync/errgroup.Group.Wait) 方法只是调用了 [`sync.WaitGroup.Wait`](https://draveness.me/golang/tree/sync.WaitGroup.Wait)，在子任务全部完成时取消 [`context.Context`](https://draveness.me/golang/tree/context.Context) 并返回可能出现的错误。

###  8.1. <a name='-1'></a>总结

[`golang/sync/errgroup.Group`](https://draveness.me/golang/tree/golang/sync/errgroup.Group) 的实现没有涉及底层和运行时包中的 API，它只是对基本同步语义进行了封装以提供更加复杂的功能。我们在使用时也需要注意下面几个问题：

- [`golang/sync/errgroup.Group`](https://draveness.me/golang/tree/golang/sync/errgroup.Group) 在出现错误或者等待结束后会调用 [`context.Context`](https://draveness.me/golang/tree/context.Context) 的 `cancel` 方法同步取消信号；
- 只有第一个出现的错误才会被返回，剩余的错误会被直接丢弃；

##  9. <a name='Semaphore'></a>Semaphore

信号量是在并发编程中常见的一种同步机制，在需要控制访问资源的进程数量时就会用到信号量，它会保证持有的计数器在 0 到初始化的权重之间波动。

- 每次获取资源时都会将信号量中的计数器减去对应的数值，在释放时重新加回来；
- 当遇到计数器大于信号量大小时，会进入休眠等待其他线程释放信号；

Go 语言的扩展包中就提供了带权重的信号量 [`golang/sync/semaphore.Weighted`](https://draveness.me/golang/tree/golang/sync/semaphore.Weighted)，我们可以按照不同的权重对资源的访问进行管理，这个结构体对外也只暴露了四个方法：

- [`golang/sync/semaphore.NewWeighted`](https://draveness.me/golang/tree/golang/sync/semaphore.NewWeighted) 用于创建新的信号量；
- [`golang/sync/semaphore.Weighted.Acquire`](https://draveness.me/golang/tree/golang/sync/semaphore.Weighted.Acquire) 阻塞地获取指定权重的资源，如果当前没有空闲资源，会陷入休眠等待；
- [`golang/sync/semaphore.Weighted.TryAcquire`](https://draveness.me/golang/tree/golang/sync/semaphore.Weighted.TryAcquire) 非阻塞地获取指定权重的资源，如果当前没有空闲资源，会直接返回 `false`；
- [`golang/sync/semaphore.Weighted.Release`](https://draveness.me/golang/tree/golang/sync/semaphore.Weighted.Release) 用于释放指定权重的资源；

[`golang/sync/semaphore.NewWeighted`](https://draveness.me/golang/tree/golang/sync/semaphore.NewWeighted) 方法能根据传入的最大权重创建一个指向 [`golang/sync/semaphore.Weighted`](https://draveness.me/golang/tree/golang/sync/semaphore.Weighted) 结构体的指针：

```go
func NewWeighted(n int64) *Weighted {
	w := &Weighted{size: n}
	return w
}

type Weighted struct {
	size    int64
	cur     int64
	mu      sync.Mutex
	waiters list.List
}
```

[`golang/sync/semaphore.Weighted`](https://draveness.me/golang/tree/golang/sync/semaphore.Weighted) 结构体中包含一个 `waiters` 列表，其中存储着等待获取资源的 Goroutine，除此之外它还包含当前信号量的上限以及一个计数器 `cur`，这个计数器的范围就是 [0, size]：

###  9.1. <a name='-1'></a>获取信号量

[`golang/sync/semaphore.Weighted.Acquire`](https://draveness.me/golang/tree/golang/sync/semaphore.Weighted.Acquire) 方法能用于获取指定权重的资源，其中包含三种不同情况：

1. 当信号量中剩余的资源大于获取的资源并且没有等待的 Goroutine 时，会直接获取信号量；
2. 当需要获取的信号量大于 [`golang/sync/semaphore.Weighted`](https://draveness.me/golang/tree/golang/sync/semaphore.Weighted) 的上限时，由于不可能满足条件会直接返回错误；
3. 遇到其他情况时会将当前 Goroutine 加入到等待列表并通过 `select` 等待调度器唤醒当前 Goroutine，Goroutine 被唤醒后会获取信号量；

```go
func (s *Weighted) Acquire(ctx context.Context, n int64) error {
	if s.size-s.cur >= n && s.waiters.Len() == 0 {
		s.cur += n
		return nil
	}

	...
	ready := make(chan struct{})
	w := waiter{n: n, ready: ready}
	elem := s.waiters.PushBack(w)
	select {
	case <-ctx.Done():
		err := ctx.Err()
		select {
		case <-ready:
			err = nil
		default:
			s.waiters.Remove(elem)
		}
		return err
	case <-ready:
		return nil
	}
}
```

另一个用于获取信号量的方法 [`golang/sync/semaphore.Weighted.TryAcquire`](https://draveness.me/golang/tree/golang/sync/semaphore.Weighted.TryAcquire) 只会非阻塞地判断当前信号量是否有充足的资源，如果有充足的资源会直接立刻返回 `true`，否则会返回 `false`：

```go
func (s *Weighted) TryAcquire(n int64) bool {
	s.mu.Lock()
	success := s.size-s.cur >= n && s.waiters.Len() == 0
	if success {
		s.cur += n
	}
	s.mu.Unlock()
	return success
}
```

因为 [`golang/sync/semaphore.Weighted.TryAcquire`](https://draveness.me/golang/tree/golang/sync/semaphore.Weighted.TryAcquire) 不会等待资源的释放，所以可能更适用于一些延时敏感、用户需要立刻感知结果的场景。

###  9.2. <a name='-1'></a>释放信号量

当我们要释放信号量时，[`golang/sync/semaphore.Weighted.Release`](https://draveness.me/golang/tree/golang/sync/semaphore.Weighted.Release) 方法会从头到尾遍历 `waiters` 列表中全部的等待者，如果释放资源后的信号量有充足的剩余资源就会通过 Channel 唤起指定的 Goroutine：

```go
func (s *Weighted) Release(n int64) {
	s.mu.Lock()
	s.cur -= n
	for {
		next := s.waiters.Front()
		if next == nil {
			break
		}
		w := next.Value.(waiter)
		if s.size-s.cur < w.n {
			break
		}
		s.cur += w.n
		s.waiters.Remove(next)
		close(w.ready)
	}
	s.mu.Unlock()
}
```

当然也可能会出现剩余资源无法唤起 Goroutine 的情况，在这时当前方法在释放锁后会直接返回。

通过对 [`golang/sync/semaphore.Weighted.Release`](https://draveness.me/golang/tree/golang/sync/semaphore.Weighted.Release) 的分析我们能发现，如果一个信号量需要的占用的资源非常多，它可能会长时间无法获取锁，这也是 [`golang/sync/semaphore.Weighted.Acquire`](https://draveness.me/golang/tree/golang/sync/semaphore.Weighted.Acquire) 引入上下文参数的原因，即为信号量的获取设置超时时间。

###  9.3. <a name='-1'></a>总结

带权重的信号量确实有着更多的应用场景，这也是 Go 语言对外提供的唯一一种信号量实现，在使用的过程中我们需要注意以下的几个问题：

- [`golang/sync/semaphore.Weighted.Acquire`](https://draveness.me/golang/tree/golang/sync/semaphore.Weighted.Acquire) 和 [`golang/sync/semaphore.Weighted.TryAcquire`](https://draveness.me/golang/tree/golang/sync/semaphore.Weighted.TryAcquire) 都可以用于获取资源，前者会阻塞地获取信号量，后者会非阻塞地获取信号量；
- [`golang/sync/semaphore.Weighted.Release`](https://draveness.me/golang/tree/golang/sync/semaphore.Weighted.Release) 方法会按照先进先出的顺序唤醒可以被唤醒的 Goroutine；
- 如果一个 Goroutine 获取了较多地资源，由于 [`golang/sync/semaphore.Weighted.Release`](https://draveness.me/golang/tree/golang/sync/semaphore.Weighted.Release) 的释放策略可能会等待比较长的时间；

##  10. <a name='SingleFlight'></a>SingleFlight

[`golang/sync/singleflight.Group`](https://draveness.me/golang/tree/golang/sync/singleflight.Group) 是 Go 语言扩展包中提供了另一种同步原语，它能够在一个服务中抑制对下游的多次重复请求。一个比较常见的使用场景是：我们在使用 Redis 对数据库中的数据进行缓存，发生缓存击穿时，大量的流量都会打到数据库上进而影响服务的尾延时。

但是 [`golang/sync/singleflight.Group`](https://draveness.me/golang/tree/golang/sync/singleflight.Group) 能有效地解决这个问题，**它能够限制对同一个键值对的多次重复请求，减少对下游的瞬时流量**。

在资源的获取非常昂贵时（例如：访问缓存、数据库），就很适合使用 [`golang/sync/singleflight.Group`](https://draveness.me/golang/tree/golang/sync/singleflight.Group) 优化服务。我们来了解一下它的使用方法：

```go
type service struct {
    requestGroup singleflight.Group
}

func (s *service) handleRequest(ctx context.Context, request Request) (Response, error) {
    v, err, _ := requestGroup.Do(request.Hash(), func() (interface{}, error) {
        rows, err := // select * from tables
        if err != nil {
            return nil, err
        }
        return rows, nil
    })
    if err != nil {
        return nil, err
    }
    return Response{
        rows: rows,
    }, nil
}
```

因为请求的哈希在业务上一般表示相同的请求，所以上述代码使用它作为请求的键。当然，我们也可以选择其他的字段作为 [`golang/sync/singleflight.Group.Do`](https://draveness.me/golang/tree/golang/sync/singleflight.Group.Do) 方法的第一个参数减少重复的请求。