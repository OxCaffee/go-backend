# Go语言中的nil

<!-- vscode-markdown-toc -->
* 1. [须知](#)
* 2. [前言](#-1)
* 3. [nil是一个预声明的标识符](#nil)
* 4. [nil不是一个关键字](#nil-1)
* 5. [nil可以表示的类型](#nil-1)
* 6. [nil没有默认类型](#nil-1)
* 7. [不同种类的nil尺寸(内存大小)不一定相同](#nil-1)
* 8. [不同类型的nil不能比较](#nil-1)
* 9. [同一个类型的两个nil也可能不能比较(易混)](#nil-1)
* 10. [两个nil可能不相等](#nil-1)
* 11. [访问nil映射值的条目不会产生panic](#nilpanic)
* 12. [range关键字后可以跟随nil通道、nil映射、nil切片和nil数组指针](#rangenilnilnilnil)
* 13. [nil型接收器调用方法会产生panic的情况](#nilpanic-1)
* 14. [若类型为T的零值可以用nil表示，则*new(T)的结果是一个T类型的nil值](#TnilnewTTnil)

<!-- vscode-markdown-toc-config
	numbering=true
	autoSave=true
	/vscode-markdown-toc-config -->
<!-- /vscode-markdown-toc -->

##  1. <a name=''></a>须知

本文章主要参考[Go中的nil](https://gfw.go101.org/article/nil.html)这篇文章并做相应的改动

##  2. <a name='-1'></a>前言

[`nil`]()是一个在Go中使用频率很高的预声明标识符。很多语言的零值都具有统一性，通用性，但是Go中的零值[`nil`]()还是有很大的限制，本文着重探究Go中的[`nil`]()特性。

##  3. <a name='nil'></a>nil是一个预声明的标识符

既然是一个预声明的标识符，我们可以不用直接定义就拿来使用。

##  4. <a name='nil-1'></a>nil不是一个关键字

上面写道，[`nil`]()是一个预声明的标识符，并不是一个关键字，**它可以被同名的内层标识符覆盖** ：

```go
func main() {
    nil := 111	// 遮盖nil
    fmt.Println(nil)	// 111
    
    var _ map[string]int = nil // 编译报错，nil为一个int值
}
```

##  5. <a name='nil-1'></a>nil可以表示的类型

[`nil`]()可以表示六大类型的零值：

* **指针类型(包括类型安全的指针和非类型安全的指针)**
* **映射类型** 
* **切片类型** 
* **函数类型** 
* **通道类型**
* **接口类型**

##  6. <a name='nil-1'></a>nil没有默认类型

go语言的其他预声明标识符都有各自的默认类型，比如[`true`]()和[`false`]()的默认类型为[`bool`]() 。

既然没有默认类型，那么就会出现不同的[`nil`]()有不同的类型，即[`nil`]()的类型是不确定的。**因此我们在代码中提供足够的信息来让编译器能够推断一个类型不确定的[`nil`]()值的期望类型** 。

```go
func main() {
	// 代码中必须提供充足的信息来让编译器推断出某个nil的类型。
	_ = (*struct{})(nil)
	_ = []int(nil)
	_ = map[int]bool(nil)
	_ = chan string(nil)
	_ = (func())(nil)
	_ = interface{}(nil)

	// 下面这一组和上面这一组等价。
	var _ *struct{} = nil
	var _ []int = nil
	var _ map[int]bool = nil
	var _ chan string = nil
	var _ func() = nil
	var _ interface{} = nil
    
    // 下面这行编译不通过，因为无法推断出nil是什么类型的
	var _ = nil
}
```

##  7. <a name='nil-1'></a>不同种类的nil尺寸(内存大小)不一定相同

不同类型的[`nil`]()内存大小不一定相同，下面看一个例子：

```go
func main() {
	var p *struct{} = nil
	fmt.Println( unsafe.Sizeof( p ) ) // 8，指针大小为1W

	var s []int = nil
	fmt.Println( unsafe.Sizeof( s ) ) // 24，切片大小为3W

	var m map[int]bool = nil
	fmt.Println( unsafe.Sizeof( m ) ) // 8，映射类型为1W

	var c chan string = nil
	fmt.Println( unsafe.Sizeof( c ) ) // 8，通道类型为1W

	var f func() = nil
	fmt.Println( unsafe.Sizeof( f ) ) // 8，函数类型为1W

	var i interface{} = nil
	fmt.Println( unsafe.Sizeof( i ) ) // 16，接口类型为2W
}
```

**如果想要探究为什么是这些数值，可以参考之前的文章[浅谈go语言中的内存对齐](/docs/Go语言内存对齐.md)** ，在这里，可以将表格再放出来：

![align](../assets/align1.png)

##  8. <a name='nil-1'></a>不同类型的nil不能比较

比如，下例中的两行中的比较均编译不通过。

```go
// error: 类型不匹配
var _ = (*int)(nil) == (*bool)(nil)
// error: 类型不匹配
var _ = (chan int)(nil) == (chan bool)(nil)
```

请阅读[Go中的值比较规则](https://gfw.go101.org/article/value-conversions-assignments-and-comparisons.html#comparison-rules)来了解哪些值可以相互比较。 类型确定的nil值也要遵循这些规则。

下面这些比较是合法的：

```go
type IntPtr *int
// 类型IntPtr的底层类型为*int。
var _ = IntPtr(nil) == (*int)(nil)

// 任何类型都实现了interface{}类型。
var _ = (interface{})(nil) == (*int)(nil)

// 一个双向通道可以隐式转换为和它的
// 元素类型一样的单项通道类型。
var _ = (chan int)(nil) == (chan<- int)(nil)
var _ = (chan int)(nil) == (<-chan int)(nil)
```

##  9. <a name='nil-1'></a>同一个类型的两个nil也可能不能比较(易混)

在Go中，**映射类型、切片类型和函数类型是不支持比较类型** 。 比较同一个不支持比较的类型的两个值（包括nil值）是非法的。 比如，下面的几个比较都编译不通过。

```go
var _ = ([]int)(nil) == ([]int)(nil)
var _ = (map[string]int)(nil) == (map[string]int)(nil)
var _ = (func())(nil) == (func())(nil)
```

但是，**映射类型、切片类型和函数类型的任何值都可以和类型不确定的裸[`nil`]()标识符比较 ** 。

```go
// 这几行编译都没问题。
var _ = ([]int)(nil) == nil
var _ = (map[string]int)(nil) == nil
var _ = (func())(nil) == nil
```

##  10. <a name='nil-1'></a>两个nil可能不相等

**如果可被比较的两个[`nil`]()值中的一个的类型为接口类型，而另一个不是，则比较结果总是[`false`]()** 。

```go
fmt.Println( (interface{})(nil) == (*int)(nil) ) // false
```

##  11. <a name='nilpanic'></a>访问nil映射值的条目不会产生panic

```go
fmt.Println( (map[string]int)(nil)["key"] ) // 0
fmt.Println( (map[int]bool)(nil)[123] )     // false
fmt.Println( (map[int]*int64)(nil)[123] )   // 
```

**访问一个nil映射将得到此映射的类型的元素类型的零值** 。

##  12. <a name='rangenilnilnilnil'></a>range关键字后可以跟随nil通道、nil映射、nil切片和nil数组指针

* 遍历nil映射和nil切片的循环步数均为零。
* 遍历一个nil数组指针的循环步数为对应数组类型的长度。 （但是，**如果此数组类型的长度不为零并且第二个循环变量未被舍弃或者忽略，则对应`for-range`循环将导致一个恐慌**。）
* 遍历一个nil通道将使当前协程永久阻塞。

```go
func main() {
	var a = []int(nil)
	var b = map[string]int(nil)
	var c = (*[5]int)(nil)
	var d = chan int(nil)

    // 循环次数为0
	for k, v := range a {
		fmt.Println("key:", k, "; value:", v)
	}

    // 循环次数为0
	for k, v := range b {
		fmt.Println("key:", k, "; value:", v)
	}

	// // 这个会触发panic
	for k, v := range c {
		fmt.Println("key:", k, "; value:", v)
	}

	// 如果不适用value，则不会panic
	for k, _ := range c {
		fmt.Println("key:", k)
	}

	// deadlock，永久阻塞
	for range d {
		fmt.Println("chan!!!")
	}
}
```

##  13. <a name='nilpanic-1'></a>nil型接收器调用方法会产生panic的情况

```go
type Slice []bool

func (s Slice) Length() int {
	return len(s)
}

func (s Slice) Modify(i int, x bool) {
	s[i] = x // panic if s is nil
}

func (p *Slice) DoNothing() {
}

func (p *Slice) Append(x bool) {
	*p = append(*p, x) // 如果p为空指针，则产生一个恐慌。
}

func main() {
	// 下面这几行中的选择器不会造成恐慌。
	_ = ((Slice)(nil)).Length
	_ = ((Slice)(nil)).Modify
	_ = ((*Slice)(nil)).DoNothing
	_ = ((*Slice)(nil)).Append

	// 这两行也不会造成恐慌。
	_ = ((Slice)(nil)).Length()
	((*Slice)(nil)).DoNothing()

	// 下面这两行都会造成恐慌。但是恐慌不是因为nil
	// 属主实参造成的。恐慌都来自于这两个方法内部的
	// 对空指针的解引用操作。
	/*
	((Slice)(nil)).Modify(0, true)
	((*Slice)(nil)).Append(true)
	*/
}
```

[`nil`]()型接收器调用方法时本身不会造成panic，但是若要修改相应的接收器数据，就会造成panic。

##  14. <a name='TnilnewTTnil'></a>若类型为T的零值可以用nil表示，则*new(T)的结果是一个T类型的nil值

```go
func main() {
	fmt.Println(*new(*int) == nil)         // true
	fmt.Println(*new([]int) == nil)        // true
	fmt.Println(*new(map[int]bool) == nil) // true
	fmt.Println(*new(chan string) == nil)  // true
	fmt.Println(*new(func()) == nil)       // true
	fmt.Println(*new(interface{}) == nil)  // true
}
```

