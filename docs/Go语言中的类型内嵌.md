# Go语言中的类型内嵌

<!-- vscode-markdown-toc -->
* 1. [须知](#)
* 2. [Go中的哪些类型可以被内嵌](#Go)
* 3. [类型内嵌允许多次内嵌吗](#-1)
* 4. [当一个结构体内嵌了另外一个类型，此结构体从内嵌的类型中获得了什么](#-1)
* 5. [选择器遮挡和碰撞](#-1)

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

##  5. <a name='-1'></a>选择器遮挡和碰撞

- 只有深度最浅的一个完整形式的选择器（并且最浅者只有一个）可以被缩写为`x.y`。 换句话说，`x.y`表示深度最浅的一个选择器。其它完整形式的选择器被此最浅者所遮挡（压制）。
- 如果有多个完整形式的选择器同时拥有最浅深度，则任何完整形式的选择器都不能被缩写为`x.y`。 我们称这些同时拥有最浅深度的完整形式的选择器发生了碰撞。

如果一个方法选择器被另一个方法选择器所遮挡，并且它们对应的方法原型是一致的，那么我们可以说第一个方法被第二个覆盖（overridden）了。

举个例子，假设`A`、`B`和`C`为三个[定义类型](https://gfw.go101.org/article/type-system-overview.html#non-defined-type)：

```go
type A struct {
	x string
}
func (A) y(int) bool {
	return false
}

type B struct {
	y bool
}
func (B) x(string) {}

type C struct {
	B
}
```

下面这段代码编译不通过，原因是选择器`v1.A.x`和`v1.B.x`的深度一样，所以它们发生了碰撞，结果导致它们都不能被缩写为`v1.x`。 同样的情况发生在选择器`v1.A.y`和`v1.B.y`身上。

```go
var v1 struct {
	A
	B
}

func f1() {
	_ = v1.x // error: 模棱两可的v1.x
	_ = v1.y // error: 模棱两可的v1.y
}
```

下面的代码编译没问题。选择器`v2.C.B.x`被另一个选择器`v2.A.x`遮挡了，所以`v2.x`实际上是选择器`v2.A.x`的缩写形式。 因为同样的原因，`v2.y`是选择器`v2.A.y`（而不是选择器`v2.C.B.y`）的缩写形式。

```go
var v2 struct {
	A
	C
}

func f2() {
	fmt.Printf("%T \n", v2.x) // string
	fmt.Printf("%T \n", v2.y) // func(int) bool
}
```

一个被遮挡或者碰撞的选择器并不妨碍更深层的选择器被提升，如下例所示中的`.M`和`.z`：

```go
package main

type x string
func (x) M() {}

type y struct {
	z byte		// y.z
}

type A struct {
	x			//A.x ==> v.A.x
}
func (A) y(int) bool {	//A.y ==> v.A.y
	return false	
}

type B struct {	//B.y	==> v.B.y
	y
}
func (B) x(string) {}	//B.x ==> v.B.x

func main() {
	var v struct {
		A	//v.A
		B	//v.B
	}
    
    // 共存在如下选择器
    // v.A.x
    // v.A.y
    // v.B.x
    // v.B.y
    // v.A.y.z
    // v.B.y.z
    
	//_ = v.x // error: 模棱两可的v.x
	//_ = v.y // error: 模棱两可的v.y
	_ = v.M // ok. <=> v.A.x.M
	_ = v.z // ok. <=> v.B.y.z
}
```

来自不同库包的两个非导出方法（或者字段）将总是被认为是两个不同的标识符，即使它们的名字完全一致。 因此，当它们的属主类型被同时内嵌在同一个结构体类型中的时候，它们绝对不会相互碰撞或者遮挡。 举个例子，下面这个含有两个库包的Go程序编译和运行都没问题。 但是，如果将其中所有出现的`m()`改为`M()`，则此程序将编译不过。 原因是`A.M`和`B.M`碰撞了，导致`c.M`为一个非法的选择器。

```go
type A struct {
	n int
}

func (a A) m() {
	fmt.Println("A", a.n)
}

type I interface {
	m()
}

func Bar(i I) {
	i.m()
}
```

