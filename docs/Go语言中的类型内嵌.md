# Go语言中的类型内嵌

## 须知

本文章主要参考[Go语言101-类型内嵌](https://gfw.go101.org/article/type-embedding.html)，并在此文基础上修改并添加个人心得。

## Go中的哪些类型可以被内嵌

1. 一个类型名[`T`]()只有在它既**不**表示一个**定义的指针类型**， 也**不**表示一个**基类型为指针类型或者接口类型的指针类型**的情况下才能被当作内嵌类型
2. 一个指针类型[`*T`]()只有在**[`T`]()为一个类型名**并且[`T`]()既**不**表示为一个**指针类型**也**不**表示为一个**接口类型**的时候才能被当作内嵌类型

更加具体的总结：

* 若[`T`]()内嵌，则[`T`]()不能为定义的指针类型，基类型为指针类型或者接口类型的指针类型(**法则一**)
* 若[`*T`]()内嵌，则[`T`]()必须为类型名，并且[`T`]()不能为指针类型或者接口类型(**法则二**)
* 若[`T`]()为非定义非指针类型，则[`T`]()不能内嵌
* 若[`T`]()为非定义类型，则[`*T`]()不能内嵌

下面列出了一些可以被或者不可以被内嵌的类型名或者别名:

```go
// 定义一些待内嵌的类型candidate
type Encoder interface {		// 定义的接口类型
    Encoder([]byte) []byte
}
type Person struct {	// 定义的结构体类型
    name string;
    age int;
}
type Alias = struct {	// 非定义的结构体类型
    name string;
    age int;
}
type AliasPtr = *struct {	// 非定义的结构体指针类型
    name string;
    age int;
}
type IntPtr *int	// 定义的指针类型
type AliasPP = *IntPtr	// 非定义的基类型为指针类型的指针类型

// 这些类型可以被内嵌
Encoder		  // 满足法则一，Encoder既不为定义的指针类型，也不为基类型为指针类型或者接口类型的指针类型
Person		  // 满足法则一，同上
*Person		  // 满足法则二，Person为类型名，且Person不为指针类型或者接口类型
Alias		  // 满足法则一，Alias虽然是被定义的，但是不是定义的指针类型
AliasPtr	  // 满足法则一，Alias虽然是指针类型，但是是非定义的指针类型
int			  // 满足法则一
*int		  // 满足法则二，int为类型名，并且int不为指针类型或者接口类型

// 这些类型不能被内嵌
AliasPP		  // 违背法则一，AliasPP的基类型为IntPtr，IntPtr是指针类型
*Encoder	  // 违背法则二，Encoder是类型名，且，Encoder是接口类型
*AliasPtr	  // 违背法则二，AliasPtr是类型名，且，AliasPtr是指针类型
IntPtr		  // 违背法则一，IntPtr是定义的指针类型
*IntPtr		  // 违背法则二。IntPtr是类型名，但是IntPtr是指针类型
*chan int	  // 基类型为非定义类型
struct {age int}	// 非定义非指针类型
map[string]int// 非定义非指针类型
func()		  // 非定义非指针类型
```

## 类型内嵌允许多次内嵌吗

* 一个结构体类型不允许有两个同名字段
* 一个结构体类型不允许有两个相同类型的匿名字段
* 一个非定义指针类型不能和它的基类型同时内嵌在一个结构体中，例如：[`int`]()和[`*int`]()

