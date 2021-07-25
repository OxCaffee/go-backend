# Go语言中的类型内嵌

<!-- vscode-markdown-toc -->
* 1. [须知](#)
* 2. [Go中的哪些类型可以被内嵌](#Go)
* 3. [类型内嵌允许多次内嵌吗](#-1)
* 4. [当一个结构体内嵌了另外一个类型，此结构体从内嵌的类型中获得了什么](#-1)

<!-- vscode-markdown-toc-config
	numbering=true
	autoSave=true
	/vscode-markdown-toc-config -->
<!-- /vscode-markdown-toc -->

##  1. <a name=''></a>须知

本文章主要参考[Go语言101-类型内嵌](https://gfw.go101.org/article/type-embedding.html)，并在此文基础上修改并添加个人心得。

##  2. <a name='Go'></a>Go中的哪些类型可以被内嵌

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

##  3. <a name='-1'></a>类型内嵌允许多次内嵌吗

* 一个结构体类型不允许有两个同名字段
* 一个结构体类型不允许有两个相同类型的匿名字段
* 一个非定义指针类型不能和它的基类型同时内嵌在一个结构体中，例如：[`int`]()和[`*int`]()

##  4. <a name='-1'></a>当一个结构体内嵌了另外一个类型，此结构体从内嵌的类型中获得了什么

```go
type Person struct {
	Name string
	Age  int
}
func (p Person) PrintName() {
	fmt.Println("Name:", p.Name)
}
func (p *Person) SetAge(age int) {
	p.Age = age
}

type Singer struct {
	Person // 通过内嵌Person类型来扩展之
	works  []string
}
```

我们利用反射来看看[`Person`]()和[`Singer`]()都有什么字段和方法：

```go
func main() {

	t := reflect.TypeOf(Singer{})
	fmt.Println("Singer有", t.NumField(), "个字段")
	fmt.Println("Singer有", t.NumMethod(), "个方法")

	for i := 0; i < t.NumField(); i++ {
		fmt.Println("Singer的第", i, "个字段为:", t.Field(i).Name)
	}

	for i := 0; i < t.NumMethod(); i++ {
		fmt.Println("Singer的第", i, "个方法为:", t.Method(i).Name)
	}

	p := reflect.TypeOf(Person{})
	fmt.Println("Person有", p.NumField(), "个字段")
	fmt.Println("Person有", p.NumMethod(), "个方法")

	for i := 0; i < p.NumField(); i++ {
		fmt.Println("Person的第", i, "个字段为:", p.Field(i).Name)
	}

	for i := 0; i < p.NumMethod(); i++ {
		fmt.Println("Person的第", i, "个方法为:", p.Method(i).Name)
	}

	tt := reflect.TypeOf(&Singer{})
	fmt.Println("*Singer有", tt.NumMethod(), "个方法")

	for i := 0; i < tt.NumMethod(); i++ {
		fmt.Println("*Singer的第", i, "个方法为:", tt.Method(i).Name)
	}
}
```

输出结果如下：

```go
Singer有 2 个字段
Singer有 1 个方法
Singer的第 0 个字段为: Person
Singer的第 1 个字段为: works
Singer的第 0 个方法为: PrintName

Person有 2 个字段
Person有 1 个方法
Person的第 0 个字段为: Name
Person的第 1 个字段为: Age
Person的第 0 个方法为: PrintName

*Singer有 2 个方法
*Singer的第 0 个方法为: PrintName
*Singer的第 1 个方法为: SetAge
```

类型[`Singer`]()和[`*Singer`]()都有一个[`PrintName`]()方法，并且类型[`*Singer`]()还有一个[`SetAge`]()方法。 但是，我们从没有为这两个类型声明过这几个方法。这几个方法从哪来的呢？

- 类型[`struct{T}`]()和[`*struct{T}`]()均将获取类型[`T`]()的所有方法。
- 类型[`*struct{T}`]()、[`struct{*T}`]()和[`*struct{*T}`]()都将获取类型[`*T`]()的所有方法。