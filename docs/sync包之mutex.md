# Go源码解读—sync.mutex

<!-- vscode-markdown-toc -->
* 1. [mutex的结构体定义](#mutex)
* 2. [mutex的状态字段](#mutex-1)
* 3. [加锁Lock](#Lock)
	* 3.1. [#lockSlow](#lockSlow)
* 4. [解锁UnLock](#UnLock)
	* 4.1. [#unlockSlow](#unlockSlow)
* 5. [问题](#)
	* 5.1. [自旋操作最多多少次](#-1)
	* 5.2. [锁的starving模式是什么情况下被设置的](#starving)
	* 5.3. [什么情况下，代表当前goroutine成功获取到了锁](#goroutine)
	* 5.4. [starving状态什么时候会被解除](#starving-1)

<!-- vscode-markdown-toc-config
	numbering=true
	autoSave=true
	/vscode-markdown-toc-config -->
<!-- /vscode-markdown-toc -->

##  1. <a name='mutex'></a>mutex的结构体定义

```go
type Mutex struct {
	state int32
	sema  uint32
}
```

`state`是一个共用的字段， **第0个** bit 标记这个`mutex`是否已被某个goroutine所拥有， 下面为了描述方便称之为`state`已加锁，或者`mutex`已加锁。 如果**第0个** bit为0, 下文称之为`state`未被锁, 此mutex目前没有被某个goroutine所拥有。

##  2. <a name='mutex-1'></a>mutex的状态字段

```go
const (
	mutexLocked = 1 << iota 	//最低一位表示mutex是否已经上锁
	mutexWoken					// 倒数第二位表示mutex是否已经被唤醒(为什么要设置这个flag，后面会具体的解释)
	mutexStarving				// 当前的mutex是否处于饥饿模式
	mutexWaiterShift = iota		// 倒数4~32位表示处于等待队列的goroutine的规模
	starvationThresholdNs = 1e6	// 如果有goroutine等待时间超过1ms(10^6ns)仍没有获得锁，那么锁就切换为饥饿模式
)
```

##  3. <a name='Lock'></a>加锁Lock

```go
func (m *Mutex) Lock() {
	// Fast path: 理想情况下，除了当前goroutine，没有其他goroutine去争抢锁
	// 理想情况下，一次CAS操作就可以获取到锁
	if atomic.CompareAndSwapInt32(&m.state, 0, mutexLocked) {
		if race.Enabled {
			race.Acquire(unsafe.Pointer(m))
		}
		return
	}
	// Slow path: 如果一次CAS没有获取到锁，就必须进行Slow path这个更复杂的逻辑去获取锁
	m.lockSlow()
}
```

###  3.1. <a name='lockSlow'></a>#lockSlow

```go
func (m *Mutex) lockSlow() {
	// 用来判断当前goroutine获取锁用了多长的时间，如果超过了starvationThresholdNs(1ms)，那么锁就进入饥饿模式
	var waitStartTime int64
	// 初始时刻，锁处于正常模式 normal mode
	starving := false
	// 本goroutine是否已经被唤醒
	awoke := false
	// 自旋次数
	iter := 0
	// 锁的初始状态
	old := m.state
	for {
		// 如果当前锁处于饥饿状态，不能发生自旋，处于饥饿状态下，所有未获取到的goroutine都会直接加入等待队列
		// runtime_canSpin则判断是否可以发生自旋，允许自旋需要满足下面的条件
		// 1. 当前处理器是多核并且GOMAXPROCS>1，因此单核情况下，自旋没有意义：持有锁的goroutine在等待持有CPU的goroutine释放
		//    CPU，而持有CPU的goroutine在等待持有锁的goroutine释放锁
		// 2. 当前锁处于正常模式
		if old&(mutexLocked|mutexStarving) == mutexLocked && runtime_canSpin(iter) {
			// 自旋过程中发现state还没有设置woken标识，则设置它的token标识，并标记为自己被唤醒
			if !awoke && old&mutexWoken == 0 && old>>mutexWaiterShift != 0 &&
				atomic.CompareAndSwapInt32(&m.state, old, old|mutexWoken) {
				awoke = true
			}
			// 执行自旋，最多执行4次自旋，自旋的本质是调用procyield 30次，每次自旋都会检查是否可以进行下一次自旋
			runtime_doSpin()
			// 自旋次数+1
			iter++
			// 获取当前锁的状态
			old = m.state
			continue
		}

		// 到了这一步， state的状态可能是：
		// 1. 锁还没有被释放，锁处于正常状态
		// 2. 锁还没有被释放， 锁处于饥饿状态
		// 3. 锁已经被释放， 锁处于正常状态
		// 4. 锁已经被释放， 锁处于饥饿状态
		//
		// 并且本goroutine的 awoke可能是true, 也可能是false (其它goroutine已经设置了state的woken标识)

		new := old
		// 如果锁处于正常模式
		if old&mutexStarving == 0 {
			// new state设置为加锁状态，并尝试CAS获取锁
			new |= mutexLocked
		}
		// 如果锁处于加锁状态或饥饿状态
		if old&(mutexLocked|mutexStarving) != 0 {
			// 将锁等待队列+1
			new += 1 << mutexWaiterShift
		}

		// 如果当前goroutine处于饥饿状态，并且锁已经被别的goroutine所占有
		// 那么直接将锁的状态设置为饥饿状态
		if starving && old&mutexLocked != 0 {
			new |= mutexStarving
		}

		// 如果本goroutine已经设置为唤醒状态，需要清除new state中的唤醒标记，因为本goroutine要么获取到了锁，那么陷入了休眠
		if awoke {
			if new&mutexWoken == 0 {
				throw("sync: inconsistent mutex state")
			}
			// 清除new state中的唤醒标记
			new &^= mutexWoken
		}

		// 通过CAS设置new state值
		if atomic.CompareAndSwapInt32(&m.state, old, new) {
			if old&(mutexLocked|mutexStarving) == 0 {
				// 如果old state状态是未被锁，并且锁不处于饥饿状态
				// 那么当前goroutine成功获取到了锁
				break // locked the mutex with CAS
			}
			// 如果之前已经等待过，那么直接插到等待队列的头部
			queueLifo := waitStartTime != 0
			if waitStartTime == 0 {
				waitStartTime = runtime_nanotime()
			}

			// 既然未能获取到锁， 那么就使用sleep原语阻塞本goroutine
			// 如果是新来的goroutine,queueLifo=false, 加入到等待队列的尾部，耐心等待
			// 如果是唤醒的goroutine, queueLifo=true, 加入到等待队列的头部
			runtime_SemacquireMutex(&m.sema, queueLifo, 1)

			// sleep之后，此goroutine被唤醒
			// 计算当前goroutine是否已经处于饥饿状态.
			starving = starving || runtime_nanotime()-waitStartTime > starvationThresholdNs
			old = m.state
			if old&mutexStarving != 0 {
				if old&(mutexLocked|mutexWoken) != 0 || old>>mutexWaiterShift == 0 {
					throw("sync: inconsistent mutex state")
				}
				delta := int32(mutexLocked - 1<<mutexWaiterShift)
				if !starving || old>>mutexWaiterShift == 1 {
					// 如果本goroutine是最后一个等待者，或者它并不处于饥饿状态，
					// 那么我们需要把锁的state状态设置为正常模式. 即退出饥饿模式
					delta -= mutexStarving
				}
				atomic.AddInt32(&m.state, delta)
				break
			}

			// 如果当前的锁是正常模式，本goroutine被唤醒，自旋次数清零，从for循环开始处重新开始
			awoke = true
			iter = 0
		} else {
			old = m.state
		}
	}

	if race.Enabled {
		race.Acquire(unsafe.Pointer(m))
	}
}
```

##  4. <a name='UnLock'></a>解锁UnLock

```go
func (m *Mutex) Unlock() {
	if race.Enabled {
		_ = m.state
		race.Release(unsafe.Pointer(m))
	}

	// 一次原子解锁
	new := atomic.AddInt32(&m.state, -mutexLocked)
	if new != 0 {
		// 一次原子解锁不成功，直接slow path
		m.unlockSlow(new)
	}
}
```

###  4.1. <a name='unlockSlow'></a>#unlockSlow

```go
func (m *Mutex) unlockSlow(new int32) {
	if (new+mutexLocked)&mutexLocked == 0 {
		throw("sync: unlock of unlocked mutex")
	}
	// 当前锁处于正常模式
	if new&mutexStarving == 0 {
		old := new
		for {
			// 如果等待队列为空或者锁没有被抢占，或者锁被唤醒，或者锁处于饥饿模式
			// 不需要抢占woken位，直接返回
			if old>>mutexWaiterShift == 0 || old&(mutexLocked|mutexWoken|mutexStarving) != 0 {
				return
			}
			// 抢占woken位，抢占成功就可以唤醒等待中的一个goroutine
			new = (old - 1<<mutexWaiterShift) | mutexWoken
			if atomic.CompareAndSwapInt32(&m.state, old, new) {
				runtime_Semrelease(&m.sema, false, 1)
				return
			}
			old = m.state
		}
	} else {
		runtime_Semrelease(&m.sema, true, 1)
	}
}
```

##  5. <a name=''></a>问题

###  5.1. <a name='-1'></a>自旋操作最多多少次

spin操作是调用`procyield`30次来达到自旋的目的，spin操作最多进行4次，如果4次spin之后仍然无法获取锁，进入等待队列等待

###  5.2. <a name='starving'></a>锁的starving模式是什么情况下被设置的

`starving`代表了锁的饥饿状态，只有一种方式可以设置`starving`值，那就是goroutine等待锁的时间超过了1ms

```go
starving = starving || runtime_nanotime()-waitStartTime > starvationThresholdNs
```

###  5.3. <a name='goroutine'></a>什么情况下，代表当前goroutine成功获取到了锁

如果当前goroutine成功执行了CAS尝试获取锁的操作，并且当前锁没有被其他goroutine抢占且没有处于饥饿状态，就代表当前goroutine获取到了锁:

```go
if atomic.CompareAndSwapInt32(&m.state, old, new) {
	if old&(mutexLocked|mutexStarving) == 0 {
		// 如果old state状态是未被锁，并且锁不处于饥饿状态
		// 那么当前goroutine成功获取到了锁
		break // locked the mutex with CAS
    }
    ...
}
```

###  5.4. <a name='starving-1'></a>starving状态什么时候会被解除

如果当前goroutine是最后一个等待获取锁的，或者当前goroutine不处于饥饿状态，那么就解除锁的饥饿状态，退出饥饿模式：

```go
if old&mutexStarving != 0 {
	if old&(mutexLocked|mutexWoken) != 0 || old>>mutexWaiterShift == 0 {
		throw("sync: inconsistent mutex state")
	}
	delta := int32(mutexLocked - 1<<mutexWaiterShift)
	if !starving || old>>mutexWaiterShift == 1 {
		// 如果本goroutine是最后一个等待者，或者它并不处于饥饿状态，
		// 那么我们需要把锁的state状态设置为正常模式. 即退出饥饿模式
		delta -= mutexStarving
	}
	atomic.AddInt32(&m.state, delta)
	break
}
```

