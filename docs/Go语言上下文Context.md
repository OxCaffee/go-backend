# Go源码阅读——context.Context

<!-- vscode-markdown-toc -->
* 1. [Context使用场景](#Context)
* 2. [Context定义](#Context-1)
* 3. [Context的函数签名](#Context-1)
* 4. [创建一个可取消的Context——#WithCancel](#ContextWithCancel)
	* 4.1. [cancel的传播——#propagateCancel](#cancelpropagateCancel)
* 5. [查找被取消的父Context——#parentCancelCtx](#ContextparentCancelCtx)
	* 5.1. [被取消的Context——cancelCtx](#ContextcancelCtx)
	* 5.2. [获取父Context的取消状态——#Done](#ContextDone)
	* 5.3. [递归获取父Context取消状态的核心逻辑——cancelCtx.#Value](#ContextcancelCtx.Value)
* 6. [取消一个Context——#cancel](#Contextcancel)
* 7. [问题](#)
	* 7.1. [Context中的done通道为nil代表什么?](#Contextdonenil)
	* 7.2. [Context是如何向上寻找第一个被取消的父Context的?](#ContextContext)
	* 7.3. [Context具体是如何执行cancel的?](#Contextcancel-1)

<!-- vscode-markdown-toc-config
	numbering=true
	autoSave=true
	/vscode-markdown-toc-config -->
<!-- /vscode-markdown-toc -->

##  1. <a name='Context'></a>Context使用场景

- 对 server 应用而言，传入的请求应该创建一个 context，接受
- 通过 `WithCancel` , `WithDeadline` , `WithTimeout` 创建的 Context 会同时返回一个 cancel 方法，这个方法必须要被执行，不然会导致 context 泄漏，这个可以通过执行 `go vet` 命令进行检查
- 应该将 `context.Context` 作为函数的第一个参数进行传递，参数命名一般为 `ctx` 不应该将 Context 作为字段放在结构体中。
- 不要给 context 传递 nil，如果你不知道应该传什么的时候就传递 `context.TODO()`
- 不要将函数的可选参数放在 context 当中，context 中一般只放一些全局通用的 metadata 数据，例如 tracing id 等等
- context 是并发安全的可以在多个 goroutine 中并发调用

##  2. <a name='Context-1'></a>Context定义

`context.Context`是一个接口的定义，定义了这个接口的一系列行为：

```go
type Context interface {
    // 返回当前 context 的结束时间，如果 ok = false 说明当前 context 没有设置结束时间
	Deadline() (deadline time.Time, ok bool)
    // 返回一个 channel，用于判断 context 是否结束，多次调用同一个 context done 方法会返回相同的 channel
	Done() <-chan struct{}
    // 当 context 结束时才会返回错误，有两种情况
    // context 被主动调用 cancel 方法取消：Canceled
    // context 超时取消: DeadlineExceeded
	Err() error
    // 用于返回 context 中保存的值, 如何查找，这个后面会讲到
	Value(key interface{}) interface{}
}
```

##  3. <a name='Context-1'></a>Context的函数签名

```go
// 创建一个带有新的 Done channel 的 context，并且返回一个取消的方法
func WithCancel(parent Context) (ctx Context, cancel CancelFunc)
// 创建一个具有截止时间的 context
// 截止时间是 d 和 parent(如果有截止时间的话) 的截止时间中更早的那一个
// 当 parent 执行完毕，或 cancel 被调用 或者 截止时间到了的时候，这个 context done 掉
func WithDeadline(parent Context, d time.Time) (Context, CancelFunc)
// 其实就是调用的 WithDeadline
func WithTimeout(parent Context, timeout time.Duration) (Context, CancelFunc)
type CancelFunc
type Context
	// 一般用于创建 root context，这个 context 永远也不会被取消，或者是 done
    func Background() Context
	// 底层和 Background 一致，但是含义不同，当不清楚用什么的时候或者是还没准备好的时候可以用它
    func TODO() Context
	// 为 context 附加值
	// key 应该具有可比性，一般不应该是 string int 这种默认类型，应该自己创建一个类型
	// 避免出现冲突，一般 key 不应该导出，如果要导出的话应该是一个接口或者是指针
    func WithValue(parent Context, key, val interface{}) Context
```

##  4. <a name='ContextWithCancel'></a>创建一个可取消的Context——#WithCancel

通过调用`context.Context#WithCancel`，我们可以定义一个可取消的上下文：

```go
func WithCancel(parent Context) (ctx Context, cancel CancelFunc) {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
    // 包装出新的 cancelContext
	c := newCancelCtx(parent)
    // 构建父子上下文的联系，确保当父 Context 取消的时候，子 Context 也会被取消
	propagateCancel(parent, &c)
	return &c, func() { c.cancel(true, Canceled) }
}
```

而`Context`的取消操作会影响所有的子`Context`，即一旦父`Context`取消了，所有的子`Context`都会产生`context canceled`错误，即所有的子`Context`也都会取消。这个过程涉及到了`Context`的取消信号的传播，那么`cancel`信号是如何在父子`Context`中传播的呢？

###  4.1. <a name='cancelpropagateCancel'></a>cancel的传播——#propagateCancel

```go
func propagateCancel(parent Context, child canceler) {
	// 首先判断 parent 能不能被取消，如果没有done通道，那么该parent无法被取消
    // 既然无法被取消，那么也不需要传播cancel，直接返回
    done := parent.Done()
	if done == nil {
		return // parent is never canceled
	}

    // 如果可以，看一下 parent 是不是已经被取消了，已经被取消的情况下直接取消 子 context
    // done存在，那么就要判断是否取消了
	select {
	case <-done:
		// 接收到done通道信息，该parent的所有子context全部cancel
		child.cancel(false, parent.Err())
		return
	default:
	}

    // 这里是向上查找可以被取消的 parent context
	if p, ok := parentCancelCtx(parent); ok {
        // 如果找到了并且没有被取消的话就把这个子 context 挂载到这个 parent context 上
        // 这样只要 parent context 取消了子 context 也会跟着被取消
		p.mu.Lock()
		if p.err != nil {
			// parent has already been canceled
			child.cancel(false, p.err)
		} else {
			if p.children == nil {
				p.children = make(map[canceler]struct{})
			}
			p.children[child] = struct{}{}
		}
		p.mu.Unlock()
	} else {
        // 如果没有找到的话就会启动一个 goroutine 去监听 parent context 的取消 channel
        // 收到取消信号之后再去调用 子 context 的 cancel 方法
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

##  5. <a name='ContextparentCancelCtx'></a>查找被取消的父Context——#parentCancelCtx

```go
func parentCancelCtx(parent Context) (*cancelCtx, bool) {
    // 这里先判断传入的 parent 是不是永远不可取消的，如果是就直接返回了
	done := parent.Done()
	if done == closedchan || done == nil {
		return nil, false
	}

    // 这里利用了 context.Value 不断向上查询值的特点，只要出现第一个可以取消的 context 的时候就会返回
    // 如果没有的话，这时候 ok 就会等于 false
	p, ok := parent.Value(&cancelCtxKey).(*cancelCtx)
	if !ok {
		return nil, false
	}
    // 这里去判断返回的 parent 的 channel 和传入的 parent 是不是同一个，是的话就返回这个 parent
	p.mu.Lock()
	ok = p.done == done
	p.mu.Unlock()
	if !ok {
		return nil, false
	}
	return p, true
}
```

###  5.1. <a name='ContextcancelCtx'></a>被取消的Context——cancelCtx

`cancelCtx`代表了被取消的`Context`，结构体定义如下：

```go
type cancelCtx struct {
	Context // 这里保存的是父 Context

	mu       sync.Mutex            // 互斥锁
	done     chan struct{}         // 关闭信号
	children map[canceler]struct{} // 保存所有的子 context，当取消的时候会被设置为 nil
	err      error
}
```

###  5.2. <a name='ContextDone'></a>获取父Context的取消状态——#Done

在 Done 方法这里采用了 懒汉式加载的方式，第一次调用的时候才会去创建这个 channel。而这个channel表示父`Context`是否被取消。

```go
func (c *cancelCtx) Done() <-chan struct{} {
	c.mu.Lock()
	if c.done == nil {
		c.done = make(chan struct{})
	}
	d := c.done
	c.mu.Unlock()
	return d
}
```

###  5.3. <a name='ContextcancelCtx.Value'></a>递归获取父Context取消状态的核心逻辑——cancelCtx.#Value

Value 方法很有意思，这里相当于是内部 `cancelCtxKey` 这个变量的地址作为了一个特殊的 key，当查询这个 key 的时候就会返回当前 context 如果不是这个 key 就会向上递归的去调用 parent context 的 Value 方法查找有没有对应的值:

```go
func (c *cancelCtx) Value(key interface{}) interface{} {
	if key == &cancelCtxKey {
		return c
	}
	return c.Context.Value(key)
}
```

##  6. <a name='Contextcancel'></a>取消一个Context——#cancel

接下来我们来看最重要的这个 cancel 方法，cancel 接收两个参数，removeFromParent 用于确认是不是把自己从 parent context 中移除，err 是 ctx.Err() 最后返回的错误信息：

```go
func (c *cancelCtx) cancel(removeFromParent bool, err error) {
	if err == nil {
		panic("context: internal error: missing cancel error")
	}
	c.mu.Lock()
	if c.err != nil {
		c.mu.Unlock()
		return // already canceled
	}
	c.err = err
    // 由于 cancel context 的 done 是懒加载的，所以有可能存在还没有初始化的情况
	if c.done == nil {
		c.done = closedchan
	} else {
		close(c.done)
	}
    // 循环的将所有的子 context 取消掉
	for child := range c.children {
		// NOTE: acquiring the child's lock while holding parent's lock.
		child.cancel(false, err)
	}
    // 将所有的子 context 和当前 context 关系解除
	c.children = nil
	c.mu.Unlock()

    // 如果需要将当前 context 从 parent context 移除，就移除掉
	if removeFromParent {
		removeChild(c.Context, c)
	}
}
```

##  7. <a name=''></a>问题

###  7.1. <a name='Contextdonenil'></a>Context中的done通道为nil代表什么?

代表该`Context`不可取消

###  7.2. <a name='ContextContext'></a>Context是如何向上寻找第一个被取消的父Context的?

通过`cancelCtx`的`Value()`方法获取该`cancelCtxKey`对应的`cancelCtx`，然后**递归**调用`Context.Value`来查找。

###  7.3. <a name='Contextcancel-1'></a>Context具体是如何执行cancel的?

* 不断向上查找第一个被取消的`Context`，如果没有找到，直接返回
* 如果找到了第一个被取消的`Context`，此时获取的是`cancelCtx`
* 将该被取消的`cancelCtx`所有子`Context`对应的`canceler`挂载它的`children`中，只需要调用`canceler.cancel`即可取消
* 调用`canceler.cancel`，该方法会调用`cancelCtx.cancel`方法
* 调用完`canceler.cancel`方法后，将`cancelCtx.children`置为`nil`
* 如果你需要，从父`Context`中移除子`Context`
