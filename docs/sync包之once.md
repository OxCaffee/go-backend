# Go并发编程——深入理解sync.Once

<!-- vscode-markdown-toc -->
* 1. [Once结构体定义](#Once)
* 2. [#Do](#Do)
	* 2.1. [错误的使用](#)
* 3. [#doSlow](#doSlow)

<!-- vscode-markdown-toc-config
	numbering=true
	autoSave=true
	/vscode-markdown-toc-config -->
<!-- /vscode-markdown-toc -->

##  1. <a name='Once'></a>Once结构体定义

```go
type Once struct {
	done uint32
	m    Mutex
}
```

done 用于判定函数是否执行，如果不为 0 会直接返回。

##  2. <a name='Do'></a>#Do

```go
func (o *Once) Do(f func()) {
	// Note: Here is an incorrect implementation of Do:
	//
	//	if atomic.CompareAndSwapUint32(&o.done, 0, 1) {
	//		f()
	//	}
	//
	// Do guarantees that when it returns, f has finished.
	// This implementation would not implement that guarantee:
	// given two simultaneous calls, the winner of the cas would
	// call f, and the second would return immediately, without
	// waiting for the first's call to f to complete.
	// This is why the slow path falls back to a mutex, and why
	// the atomic.StoreUint32 must be delayed until after f returns.

	if atomic.LoadUint32(&o.done) == 0 {
		// Outlined slow-path to allow inlining of the fast-path.
		o.doSlow(f)
	}
}
```

在上面官方的说法是，使用`atomic.CompareAndSwapUint32(...)`是错误的用法。**经过测试，这种写法，虽然能保证f()最多执行一次，但是无法保证f()最少执行一次，因为如果一个goroutine先行抢占了`o.done`，其他goroutine不会等当前f()执行成功，而是会立即返回** 。

###  2.1. <a name=''></a>错误的使用

```go
func (o *MyOnce) Do(f func()) {
	if atomic.CompareAndSwapUint32(&o.done, 0, 1) {
		f()
	}
}
```

##  3. <a name='doSlow'></a>#doSlow

```go
func (o *Once) doSlow(f func()) {
	o.m.Lock()
	defer o.m.Unlock()
	if o.done == 0 {
        // 这里只有f()成功执行完成，才会设置o.done为1
		defer atomic.StoreUint32(&o.done, 1)
		f()
	}
}
```

