# Go语言上下文Context

<!-- vscode-markdown-toc -->
* 1. [前言](#)
* 2. [如何使用Context同步信号](#Context)
	* 2.1. [WithCancel()](#WithCancel)
	* 2.2. [WithDeadline()](#WithDeadline)
	* 2.3. [WithTimeout()](#WithTimeout)
* 3. [默认上下文](#-1)
* 4. [Context层级关系](#Context-1)
* 5. [取消信号](#-1)
* 6. [值传递](#-1)
* 7. [总结](#-1)

<!-- vscode-markdown-toc-config
	numbering=true
	autoSave=true
	/vscode-markdown-toc-config -->
<!-- /vscode-markdown-toc -->

##  1. <a name=''></a>前言

`contex.Context`是go 1.7之后引入的一个接口，用来设置截止日期，同步信号，传递请求值相关的结构体，和Goroutine有着十分重要的关系。

该接口定义了4个需要实现的方法：

1. `Deadline` ：返回该`context.Context`被取消的时间，也就是该`context.Context`的截止日期
2. `Done` ：返回一个Channel，该Channel会在任务完成之后或者上下文被取消的时候关闭，多次调用`Done`会返回同一个Channel
3. `Err` ：返回该`context.Context`被关闭的原因，它只会在`Done`方法对应的Channel被关闭时返回非空值
   * 如果`context.Context`被取消，它会返回`Canceled`错误
   * 如果`context.Context`超时，它会返回`DeadlineExceeded`错误
4. `Value` ：从 `context.Context`中获取键对应的值，对于同一个上下文来说，多次调用 `Value` 并传入相同的 `Key` 会返回相同的结果，该方法可以用来传递请求特定的数据；

```go
type Context interface {
	Deadline() (deadline time.Time, ok bool)
	Done() <-chan struct{}
	Err() error
	Value(key interface{}) interface{}
}
```

`context`包中提供的`context.Background`, `context.TODO`, `context.WithDeadline`, `context.WithValue` 会返回该接口的私有实现结构体。

在 Goroutine 构成的树形结构中对信号进行同步以减少计算资源的浪费是 `context.Context` 的最大作用。Go 服务的每一个请求都是通过单独的 Goroutine 处理的。HTTP/RPC 请求的处理器会启动新的 Goroutine 访问数据库和其他服务。

如下图所示，我们可能会创建多个 Goroutine 来处理一次请求，而 `context.Context`的作用是在不同 Goroutine 之间同步请求特定数据、取消信号以及处理请求的截止日期。

<div align=center><img src="../assets/context1.png"/></div>

每一个 `context.Context` 都会从最顶层的 Goroutine 一层一层传递到最下层。`context.Context` 可以在上层 Goroutine 执行出现错误时，将信号及时同步给下层，就可以在下层及时停掉无用的工作以减少额外资源的消耗：

##  2. <a name='Context'></a>如何使用Context同步信号

###  2.1. <a name='WithCancel'></a>WithCancel()

基本使用语法如下：

```go
func WithCancel(parent Context) (ctx Context, cancel CancelFunc)
```

`WithCancel`返回一个父`context`的副本，该副本含有一个新的`Done`Channel。返回的`Done`Channel会在`cancel`方法被调用或父`context`的`Done`Channel被关闭的时候关闭。

下面通过一个例子看看`WithCancel`的用法：

```go
package main

import (
	"context"
	"fmt"
)

func main() {
    // gen函数负责不断产生数字并放到dst channel中
    // 一旦检测到context的Done被调用(即被取消，则停止生产，防止goroutine继续生产导致泄露)
	gen := func(ctx context.Context) <-chan int {
		n := 1
		dst := make(chan int)

		go func() {
			for {
				select {
				case <-ctx.Done():
					return	// 防止泄露
				case dst <- n:
					n++
				}
			}
		}()
		return dst
	}

    // 创建一个WithCancel
	ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()

    // 我们测试1000个数字，也就是说，不进行干预的时候，应该会打印1-1000
	go func() {
		for n := range gen(ctx) {
			fmt.Println(n)
			if n == 1000 {
				break
			}
		}
	}()

    // 这里休眠10ms，本来打印1000个数字远远不止10ms，所以我们将在打印完成之前调用cancel()
	<-time.After(10 * time.Millisecond)

    // 这里调用了cancel，context被取消，channel被关闭
	go func() {
		fmt.Println("cancel is called")
		cancel()
	}()
```

###  2.2. <a name='WithDeadline'></a>WithDeadline()

基本用法如下：

```go
func WithDeadline(parent Context, d time.Time) (Context, CancelFunc)
```

下面的例子中，我们设置了一个有固定截止时间的`context`，并在它的截至时间内打印出一个overslept，在截至时间到的时候再打印出`Err`：

```go
func main() {
	const duration = 10 * time.Millisecond
	deadline := time.Now().Add(duration)

	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	go func(ctx context.Context) {
		select {
		case <-time.After(1 * time.Microsecond):
			fmt.Println("overslept")
		case <-ctx.Done():
			fmt.Println(ctx.Err())
			return
		}
	}(ctx)

	<-time.After(1 * time.Second)
}
```

###  2.3. <a name='WithTimeout'></a>WithTimeout()

基本用法如下：

```go
func WithTimeout(parent Context, timeout time.Duration) (Context, CancelFunc)
```

基本用法和`WithDeadline`差不多，只不过不需要设置截止日期，而是用超时时间代替

```go
const shortDuration = 1 * time.Millisecond

func main() {
	// Pass a context with a timeout to tell a blocking function that it
	// should abandon its work after the timeout elapses.
	ctx, cancel := context.WithTimeout(context.Background(), shortDuration)
	defer cancel()

	select {
	case <-time.After(1 * time.Second):
		fmt.Println("overslept")
	case <-ctx.Done():
		fmt.Println(ctx.Err()) // prints "context deadline exceeded"
	}
}
```

##  3. <a name='-1'></a>默认上下文

`context` 包中最常用的方法还是 `context.Background`、`context.TODO`，这两个方法都会返回预先初始化好的私有变量 `background` 和 `todo`，它们会在同一个 Go 程序中被复用：

```go
func Background() Context {
	return background
}

func TODO() Context {
	return todo
}
```

这两个私有变量都是通过 `new(emptyCtx)` 语句初始化的，它们是指向私有结构体 `context.emptyCtx` 的指针，这是最简单、最常用的上下文类型：

```go
type emptyCtx int

func (*emptyCtx) Deadline() (deadline time.Time, ok bool) {
	return
}

func (*emptyCtx) Done() <-chan struct{} {
	return nil
}

func (*emptyCtx) Err() error {
	return nil
}

func (*emptyCtx) Value(key interface{}) interface{} {
	return nil
}
```

##  4. <a name='Context-1'></a>Context层级关系

从源代码来看，[`context.Background`](https://draveness.me/golang/tree/context.Background) 和 [`context.TODO`](https://draveness.me/golang/tree/context.TODO) 也只是互为别名，没有太大的差别，只是在使用和语义上稍有不同：

- [`context.Background`](https://draveness.me/golang/tree/context.Background) 是上下文的默认值，所有其他的上下文都应该从它衍生出来；
- [`context.TODO`](https://draveness.me/golang/tree/context.TODO) 应该仅在不确定应该使用哪种上下文时使用；

在多数情况下，如果当前函数没有上下文作为入参，我们都会使用 [`context.Background`](https://draveness.me/golang/tree/context.Background) 作为起始的上下文向下传递。

<div align=center><img src="../assets/context2.png"/></div>

##  5. <a name='-1'></a>取消信号

`context.WithCancel`函数能够从 `context.Context` 中衍生出一个新的子上下文并返回用于取消该上下文的函数。一旦我们执行返回的取消函数，当前上下文以及它的子上下文都会被取消，所有的 Goroutine 都会同步收到这一取消信号。

```go
func main() {
	parentCtx, cancel1 := context.WithCancel(context.Background())
	// 这里我们不设置defer cancel1()，而是自己手动cancel

	childCtx, cancel2 := context.WithCancel(parentCtx)
	defer cancel2()

	go func(pCtx context.Context) {
		select {
		case <-pCtx.Done():
			fmt.Println("parent context", pCtx.Err())
			return
		}
	}(parentCtx)

	<-time.After(1 * time.Second)

	go func(cCtx context.Context) {
		select {
		case <-cCtx.Done():
			fmt.Println("child context", cCtx.Err())
		}
	}(childCtx)

	<-time.After(time.Second)

	// 此时父context被取消，接着会打印子context被取消
	cancel1()

	<-time.After(time.Second)
}
```

我们直接从 `context.WithCancel` 函数的实现来看它到底做了什么：

```go
func WithCancel(parent Context) (ctx Context, cancel CancelFunc) {
	c := newCancelCtx(parent)
	propagateCancel(parent, &c)
	return &c, func() { c.cancel(true, Canceled) }
}
```

- `context.newCancelCtx`将传入的上下文包装成私有结构体 `context.cancelCtx`；
- `context.propagateCancel`会构建父子上下文之间的关联，当父上下文被取消时，子上下文也会被取消：

```go
func propagateCancel(parent Context, child canceler) {
	done := parent.Done()
	if done == nil {
		return // 父上下文不会触发取消信号
	}
	select {
	case <-done:
		child.cancel(false, parent.Err()) // 父上下文已经被取消
		return
	default:
	}

	if p, ok := parentCancelCtx(parent); ok {
		p.mu.Lock()
		if p.err != nil {
			child.cancel(false, p.err)
		} else {
			p.children[child] = struct{}{}
		}
		p.mu.Unlock()
	} else {
		go func() {
			select {
			case <-parent.Done():
				child.cancel(false, parent.Err())
			case <-child.Done():
			}
		}()
	}
}
```

上述函数总共与父上下文相关的三种不同的情况：

1. 当 `parent.Done() == nil`，也就是 `parent` 不会触发取消事件时，当前函数会直接返回；
2. 当`child`的继承链包含可以取消的上下文时，会判断`parent`是否已经触发了取消信号；
   - 如果已经被取消，`child` 会立刻被取消；
   - 如果没有被取消，`child` 会被加入 `parent` 的 `children` 列表中，等待 `parent` 释放取消信号；
3. 当父上下文是开发者自定义的类型、实现了`context.Context`接口并在`Done()`方法中返回了非空的管道时；
   1. 运行一个新的 Goroutine 同时监听 `parent.Done()` 和 `child.Done()` 两个 Channel；
   2. 在 `parent.Done()` 关闭时调用 `child.cancel` 取消子上下文；

`context.propagateCancel`的作用是在 `parent` 和 `child` 之间同步取消和结束的信号，保证在 `parent` 被取消时，`child` 也会收到对应的信号，不会出现状态不一致的情况。

`context.cancelCtx` 实现的几个接口方法也没有太多值得分析的地方，该结构体最重要的方法是 `context.cancelCtx.cancel`，该方法会关闭上下文中的 Channel 并向所有的子上下文同步取消信号：

```go
func (c *cancelCtx) cancel(removeFromParent bool, err error) {
	c.mu.Lock()
	if c.err != nil {
		c.mu.Unlock()
		return
	}
	c.err = err
	if c.done == nil {
		c.done = closedchan
	} else {
		close(c.done)
	}
	for child := range c.children {
		child.cancel(false, err)
	}
	c.children = nil
	c.mu.Unlock()

	if removeFromParent {
		removeChild(c.Context, c)
	}
}
```

除了 `context.WithCancel`之外，`context` 包中的另外两个函数 `context.WithDeadline` 和 `context.WithTimeout` 也都能创建可以被取消的计时器上下文 `context.timerCtx`：

```go
func WithTimeout(parent Context, timeout time.Duration) (Context, CancelFunc) {
	return WithDeadline(parent, time.Now().Add(timeout))
}

func WithDeadline(parent Context, d time.Time) (Context, CancelFunc) {
	if cur, ok := parent.Deadline(); ok && cur.Before(d) {
		return WithCancel(parent)
	}
	c := &timerCtx{
		cancelCtx: newCancelCtx(parent),
		deadline:  d,
	}
	propagateCancel(parent, c)
	dur := time.Until(d)
	if dur <= 0 {
		c.cancel(true, DeadlineExceeded) // 已经过了截止日期
		return c, func() { c.cancel(false, Canceled) }
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err == nil {
		c.timer = time.AfterFunc(dur, func() {
			c.cancel(true, DeadlineExceeded)
		})
	}
	return c, func() { c.cancel(true, Canceled) }
}
```

[`context.WithDeadline`](https://draveness.me/golang/tree/context.WithDeadline) 在创建 [`context.timerCtx`](https://draveness.me/golang/tree/context.timerCtx) 的过程中判断了父上下文的截止日期与当前日期，并通过 [`time.AfterFunc`](https://draveness.me/golang/tree/time.AfterFunc) 创建定时器，当时间超过了截止日期后会调用 [`context.timerCtx.cancel`](https://draveness.me/golang/tree/context.timerCtx.cancel) 同步取消信号。

[`context.timerCtx`](https://draveness.me/golang/tree/context.timerCtx) 内部不仅通过嵌入 [`context.cancelCtx`](https://draveness.me/golang/tree/context.cancelCtx) 结构体继承了相关的变量和方法，还通过持有的定时器 `timer` 和截止时间 `deadline` 实现了定时取消的功能：

```go
type timerCtx struct {
	cancelCtx
	timer *time.Timer // Under cancelCtx.mu.

	deadline time.Time
}

func (c *timerCtx) Deadline() (deadline time.Time, ok bool) {
	return c.deadline, true
}

func (c *timerCtx) cancel(removeFromParent bool, err error) {
	c.cancelCtx.cancel(false, err)
	if removeFromParent {
		removeChild(c.cancelCtx.Context, c)
	}
	c.mu.Lock()
	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
	}
	c.mu.Unlock()
}
```

[`context.timerCtx.cancel`](https://draveness.me/golang/tree/context.timerCtx.cancel) 方法不仅调用了 [`context.cancelCtx.cancel`](https://draveness.me/golang/tree/context.cancelCtx.cancel)，还会停止持有的定时器减少不必要的资源浪费。

##  6. <a name='-1'></a>值传递

在最后我们需要了解如何使用上下文传值，[`context`](https://github.com/golang/go/tree/master/src/context) 包中的 [`context.WithValue`](https://draveness.me/golang/tree/context.WithValue) 能从父上下文中创建一个子上下文，传值的子上下文使用 [`context.valueCtx`](https://draveness.me/golang/tree/context.valueCtx) 类型：

```go
func WithValue(parent Context, key, val interface{}) Context {
	if key == nil {
		panic("nil key")
	}
	if !reflectlite.TypeOf(key).Comparable() {
		panic("key is not comparable")
	}
	return &valueCtx{parent, key, val}
}
```

[`context.valueCtx`](https://draveness.me/golang/tree/context.valueCtx) 结构体会将除了 `Value` 之外的 `Err`、`Deadline` 等方法代理到父上下文中，它只会响应 [`context.valueCtx.Value`](https://draveness.me/golang/tree/context.valueCtx.Value) 方法，该方法的实现也很简单：

```go
type valueCtx struct {
	Context
	key, val interface{}
}

func (c *valueCtx) Value(key interface{}) interface{} {
	if c.key == key {
		return c.val
	}
	return c.Context.Value(key)
}
```

如果 [`context.valueCtx`](https://draveness.me/golang/tree/context.valueCtx) 中存储的键值对与 [`context.valueCtx.Value`](https://draveness.me/golang/tree/context.valueCtx.Value) 方法中传入的参数不匹配，就会从父上下文中查找该键对应的值直到某个父上下文中返回 `nil` 或者查找到对应的值。

##  7. <a name='-1'></a>总结

Go 语言中的 [`context.Context`](https://draveness.me/golang/tree/context.Context) 的主要作用还是在多个 Goroutine 组成的树中同步取消信号以减少对资源的消耗和占用，虽然它也有传值的功能，但是这个功能我们还是很少用到。

在真正使用传值的功能时我们也应该非常谨慎，使用 [`context.Context`](https://draveness.me/golang/tree/context.Context) 传递请求的所有参数一种非常差的设计，比较常见的使用场景是传递请求对应用户的认证令牌以及用于进行分布式追踪的请求 ID。

使用context比并发原语或者扩展语句更加的简单。主要是操作的时候减少了deadlock的机会，不过也会引入狗皮膏药一样的ctx，不过总体来看context包确实能解决好了控制goroutine树这个问题。另外不要使用withvalue来轻易传递值，因为它寻址的时候是递归的往父树上去寻找，很慢的。