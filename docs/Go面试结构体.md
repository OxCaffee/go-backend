# Go面试——结构体

## 有关结构体值的类型转换问题

两个类型分别为[`S1`]()和[`S2`]()的结构体**值**只有在底层类型相同的时候(忽略掉字段标签)的情况下才能相互转化为对方的类型。**特别地，如果[`S1`]()和[`S2`]()底层类型相同(要忽略掉字段标签)并且只要它们其中一个为非定义类型，则可以进行隐式转换** 。

例如，下面有五个结构体类型：[`S1`]()、[`S2`]()、[`S3`]()、[`S4`]()、[`S0`]()：

* 类型[`S0`]()的值不能转化为其他四个类型中任意的一个，原因是它和另外四个类型的对应字段名不同
* 类型[`S1`]()、[`S2`]()、[`S3`]()、[`S4`]()的任意两个值可以转换为对方的类型

特别地，

* [`S2`]()表示的类型的值可以被隐式地转换为类型[`S3`]()，反之亦然。
* [`S2`]()表示的类型的值可以被隐式的转换为类型[`S4`]()，反之亦然。

但是，

* [`S2`]()表示的类型的值必须被显式地转换为类型[`S1`]()，反之亦然。
* [`S3`]()的值必须被显式地转换为[`S4`]()，反之亦然。

```go
type S0 struct {
	y int "foo"
	x bool
}

type S1 = struct { // S1是一个非定义类型
	x int "foo"
	y bool
}

type S2 = struct { // S2也是一个非定义类型
	x int "bar"
	y bool
}

type S3 S2 // S3是一个定义类型。
type S4 S3 // S4是一个定义类型。
// 如果不考虑字段标签，S3（S4）和S1的底层类型一样。
// 如果考虑字段标签，S3（S4）和S1的底层类型不一样。

var v0, v1, v2, v3, v4 = S0{}, S1{}, S2{}, S3{}, S4{}
func f() {
	v1 = S1(v2); v2 = S2(v1)
	v1 = S1(v3); v3 = S3(v1)
	v1 = S1(v4); v4 = S4(v1)
	v2 = v3; v3 = v2 // 这两个转换可以是隐式的
	v2 = v4; v4 = v2 // 这两个转换也可以是隐式的
	v3 = S3(v4); v4 = S4(v3)
}
```

**事实上，两个结构体值只有在它们可以相互隐式转换为对方的类型的时候才能相互赋值和比较** 。

## 匿名结构体类型可以使用在结构体字段声明中吗

匿名结构体类型允许出现在结构体字段声明中。匿名结构体类型也允许出现在组合字面量中。

```go
var aBook = struct {
	author struct { // 此字段的类型为一个匿名结构体类型
		firstName, lastName string
		gender              bool
	}
	title string
	pages int
}{
	author: struct {
		firstName, lastName string
		gender              bool
	}{
		firstName: "Mark",
		lastName: "Twain",
	}, // 此组合字面量中的类型为一个匿名结构体类型
	title: "The Million Pound Note",
	pages: 96,
}  
```

## 结构体可以内嵌的条件

- 一个类型名`T`只有在它既不表示一个定义的指针类型也不表示一个基类型为指针类型或者接口类型的指针类型的情况下才可以被用作内嵌字段。
- 一个指针类型`*T`只有在`T`为一个类型名并且`T`既不表示一个指针类型也不表示一个接口类型的时候才能被用作内嵌字段。

简而言之就是：

* [`T`]()作内嵌字段时，[`T`]()不能表示为**定义的指针类型，基类型为指针类型或者接口类型的指针类型**
* [`*T`]()作内嵌字段时，[`T`]()必须为类型名并且[`T`]()不为接口类型或者指针类型

```go
type Encoder interface {Encode([]byte) []byte}
type Person struct {name string; age int}
type Alias = struct {name string; age int}
type AliasPtr = *struct {name string; age int}
type IntPtr *int
type AliasPP = *IntPtr

// 这些类型或别名都可以被内嵌。
Encoder
Person
*Person
Alias
*Alias
AliasPtr
int
*int

// 这些类型或别名都不能被内嵌。
AliasPP          // 基类型为一个指针类型
*Encoder         // 基类型为一个接口类型
*AliasPtr        // 基类型为一个指针类型
IntPtr           // 定义的指针类型
*IntPtr          // 基类型为一个指针类型
*chan int        // 基类型为一个非定义类型
struct {age int} // 非定义非指针类型
map[string]int   // 非定义非指针类型
[]int64          // 非定义非指针类型
func()           // 非定义非指针类型
```

## 当一个结构体类型内嵌了另一个类型，此结构体类型是否获取了被内嵌类型的字段和方法

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

func main() {
	t := reflect.TypeOf(Singer{}) // the Singer type
	fmt.Println(t, "has", t.NumField(), "fields:")
	for i := 0; i < t.NumField(); i++ {
		fmt.Print(" field#", i, ": ", t.Field(i).Name, "\n")
	}
	fmt.Println(t, "has", t.NumMethod(), "methods:")
	for i := 0; i < t.NumMethod(); i++ {
		fmt.Print(" method#", i, ": ", t.Method(i).Name, "\n")
	}

	pt := reflect.TypeOf(&Singer{}) // the *Singer type
	fmt.Println(pt, "has", pt.NumMethod(), "methods:")
	for i := 0; i < pt.NumMethod(); i++ {
		fmt.Print(" method#", i, ": ", pt.Method(i).Name, "\n")
	}
}
```

输出结果：

```go
main.Singer has 2 fields:
 field#0: Person
 field#1: works
main.Singer has 1 methods:
 method#0: PrintName
*main.Singer has 2 methods:
 method#0: PrintName
 method#1: SetAge
```

类型`Singer`和`*Singer`都有一个`PrintName`方法，并且类型`*Singer`还有一个`SetAge`方法。 但是，我们从没有为这两个类型声明过这几个方法。这几个方法从哪来的呢？

事实上，假设结构体类型`S`内嵌了一个类型（或者类型别名）`T`，并且此内嵌是合法的，

- 对内嵌类型`T`的每一个方法，如果此方法对应的选择器既不和其它选择器碰撞也未被其它选择器遮挡，则编译器将会隐式地为结构体类型`S`声明一个同样原型的方法。 继而，编译器也将为指针类型`*S`[隐式声明](https://gfw.go101.org/article/method.html#implicit-pointer-methods)一个相应的方法。
- 对类型`*T`的每一个方法，如果此方法对应的选择器既不和其它选择器碰撞也未被其它选择器遮挡，则编译器将会隐式地为类型`*S`声明一个同样原型的方法。

简单说来，

- 类型`struct{T}`和`*struct{T}`均将获取类型`T`的所有方法。
- 类型`*struct{T}`、`struct{*T}`和`*struct{*T}`都将获取类型`*T`的所有方法。