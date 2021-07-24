# Go语言中的Happens-Before原则

<!-- vscode-markdown-toc -->
* 1. [前言](#)
* 2. [什么是happens-before原则](#happens-before)
* 3. [ happens-before原则有什么用](#happens-before-1)
* 4. [happens-before原则具体内容](#happens-before-1)
* 5. [Go语言中的happens-before](#Gohappens-before)
	* 5.1. [单协程的happens-before原则](#happens-before-1)
	* 5.2. [初始化方法的happens-before原则](#happens-before-1)
	* 5.3. [协程生命周期的happens-before原则](#happens-before-1)
	* 5.4. [通道中的happens-before原则](#happens-before-1)
	* 5.5. [锁的happens-before原则](#happens-before-1)
	* 5.6. [Once的happens-before原则](#Oncehappens-before)

<!-- vscode-markdown-toc-config
	numbering=true
	autoSave=true
	/vscode-markdown-toc-config -->
<!-- /vscode-markdown-toc -->

##  1. <a name=''></a>前言

今天阅读了Golang内存模型的官方文档，虽然不长，但是需要反复阅读来理解。整个Golang的内存模型基本上就是围绕[`happens-before`]()原则来阐述的。

**了解go语言中的[`happens-before`]()原则，可以帮助我们寻找并发程序不确定性中的确定性** 。

##  2. <a name='happens-before'></a>什么是happens-before原则

你可以把[`happens-before`]()看作一种特殊的比较运算，就好像`>`、`<`一样。对应的，还有[`happens-after`]() ，它们之间的关系也好像`>`、`<`一样：

```markdown
如果A happens-before B, 那么B happens-after A
```

如果存在下面一种情况：

* A 既不 [`happens-before`]() B
* B 也不 [`happens-before`]() A

那么我们就称A和B处于[`happen concurrently`]() ，即并发

[`happens-before`]() 原则在应用程序执行方面，就表现为**语句的执行顺序问题** ，如果A [`happens-before`]() B，那么在执行时期 A语句一定是先于B语句执行的。

##  3. <a name='happens-before-1'></a> happens-before原则有什么用

对于编程者而言，[`happens-before`]() 能够帮助我们**捋清两个并发读写之间的关系** 。应用程序中的并发问题本质上还是读写顺序的问题，以及是否能够正确的读写。当我们写完源代码时，在运行代码文件之前，要经过编译的步骤，然而编译器常常会有**指令重排序的优化** ，当我们同时定义了[`a=3;b=4`]() 时，在机器层面，有可能b的定义先于a，**这是因为编译器为了优化执行效率可能对指令进行重排序** 。再比如，一条自增操作 [`a += 3`]() 实际上包含了三个原子操作(取a，a加减，写回a) ，那么整个过程就不是原子化的了，就意味着随时有其他“入侵者”侵入修改数据，造成最后程序得到一个预期之外的数据。

整个并发编程的世界充满了不确定性，我们不知道每次读取的变量到底是不是及时，但是[`happens-before`]()原则能够帮助我们去理解并发程序的本质。]

##  4. <a name='happens-before-1'></a>happens-before原则具体内容

1. **一次对于变量[`v`]()的读取操作[`r`]()能够观测到一个对于变量[`v`]()的修改操作[`w`]()的条件满足** ：
   1. [`w`]() 的发生早于[`r`]()
   2. 在[`w`]() 和 [`r`]() 之间没有更早的其他修改操作 [`w'`]()
2. **为了确保[`r`]()能观测到特定的[`w`]()，需满足以下条件** ：
   1. [`w`]()的发生早于[`r`]()
   2. 其他的[`w'`]()要么发生早于[`w`]()，要么发生晚于[`r`]()

实际上第2条准则是对第一条的更为具体准确的阐述，上述翻译自Go官方文档。

##  5. <a name='Gohappens-before'></a>Go语言中的happens-before

###  5.1. <a name='happens-before-1'></a>单协程的happens-before原则

**单协程中，书写在前的代码[`happens-before`]()书写在后的代码**  。

先来看一个例子：

```go
a := 3	// (1)
b := 4  // (2)
```

我们在单协程中创建了两个变量a和b，当发生指令重排序的时候，b的创建可能先于a，但是这并不能否认单协程中的[`happens-before`]()原则，因为a和b的创建顺序互不相干，所以即使发生了指令重排序，[`happens-before`]()原则依然存在。

###  5.2. <a name='happens-before-1'></a>初始化方法的happens-before原则

每个go文件都有一个[`init`]()方法，用于执行初始化逻辑。**当我们开始执行main方法时，go会现在一个goroutine中做初始化工作，即调用[`init`]()** , 关于[`init`]()方法的[`happens-before`]()原则如下：

* 若A包导入了B包，此时B包的[`init`]()方法[`happens-before`]() A包的[`init`]()方法
* 所有的[`init`]()都[`happens-before`]()程序中的[`main`]()方法

###  5.3. <a name='happens-before-1'></a>协程生命周期的happens-before原则

* goroutine的创建[`happens-before`]()其执行
* goroutine的完成**不保证**[`happens-before`]()任何代码

###  5.4. <a name='happens-before-1'></a>通道中的happens-before原则

通道channel是go语言中用于goroutine之间通信的主要渠道，因此理解通道之间的happens-before规则也至关重要。

* **对于无缓冲通道，向通道中发送数据[`happens-before`]()从通道中接收数据**

```go
var c = make(chan int, 10)
var a string

func f() {
    a = "hello, world" // (1)
    c <- 0 // (2)
}

func main() {
    go f() // (3)
    <-c // (4)
    fmt.Println(a) // (5)
}
```

`c`是一个缓冲通道，因此向通道发送数据`happens-before`从通道接收到数据，也就是(2) `happens-before` (4)，再结合自然执行规则以及传递性不难推导出(1) happens-before (5)，也就是打印的结果保证是"hello world"。

* **对于无缓冲通道，从通道中接收数据[`happens-before`]()向通道中发送数据**

我们可以将上述例子稍微调整下：

```go
var c = make(chan int)
var a string

func f() {
    a = "hello, world" // (1)
    <- c // (2)
}

func main() {
    go f() // (3)
    c <- 10 // (4)
    fmt.Println(a) // (5)
}
```

对于无缓冲通道，(2) `happens-before` (4)，再根据传递性，(1) `happens-before` (5)，因此依然可以保证打印的结果是"hello world"。

###  5.5. <a name='happens-before-1'></a>锁的happens-before原则

* **对互斥锁实例调用n次Unlock [`happens-before`]()调用Lock m次，只要n < m**

```go
var l sync.Mutex
var a string

func f() {
    a = "hello, world" // (1)
    l.Unlock() // (2)
}

func main() {
    l.Lock() // (3)
    go f() // (4)
    l.Lock() // (5)
    print(a) // (6)
}
```

上面调用了`Unlock`一次，`Lock`两次，因此(2) `happens-before` (5)，从而(1) `happens-before` (6)

* **对写锁的释放[`happens-after`]()对读锁的获取**

* **对读锁的释放[`happens-before`]()下一次写锁的获取**

###  5.6. <a name='Oncehappens-before'></a>Once的happens-before原则

sync中还提供了一个`Once`的数据结构，用于控制并发编程中只执行一次的逻辑，例如：

```go
var a string
var once sync.Once

func setup() {
   a = "hello, world"
   fmt.Println("set up")
}

func doprint() {
   once.Do(setup)
   fmt.Println(a)
}

func twoprint() {
    go doprint()
    go doprint()
}
```

会打印"hello, world"两次和"set up"一次。`Once`的`happens-before`规则也很直观：

**Once的原则是：第一次执行[`Once.Do`]() [`happens-before`]()其余的[`Once.Do`]()**

