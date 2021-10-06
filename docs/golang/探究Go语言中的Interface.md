# 探究Go语言中的interface

<!-- vscode-markdown-toc -->
* 1. [前言](#)
* 2. [探究的目标](#-1)
* 3. [interface的组成部分](#interface)
	* 3.1. [接口的方法数据——fun数组](#fun)
	* 3.2. [接口的类型描述——interfacetype](#interfacetype)
	* 3.3. [类型的运行时表示—— _type](#_type)
* 4. [值接收者和指针接收者](#-1)
	* 4.1. [两者分别在何时使用](#-1)
* 5. [有关nil和interface](#nilinterface)
* 6. [interface的创建过程(太难了，慢慢写)](#interface-1)

<!-- vscode-markdown-toc-config
	numbering=true
	autoSave=true
	/vscode-markdown-toc-config -->
<!-- /vscode-markdown-toc -->

##  1. <a name=''></a>前言

Go语言并没有设计诸如虚函数，纯虚函数，继承，多重继承等概念，但是它通过接口非常优雅地实现了支持面向对象地特性。多态是一种**运行期**的行为，有下面几个特点：

* 一种类型具有多种类型的能力
* 允许不同的对象对同一个消息做出灵活的反应
* 以一种通用的方式对待使用的对象
* 非动态语言必须通过继承和接口的方式来实现

Golang通过接口`interfacc`的方式实现了面向对象的特性，即**鸭子类型** ：

```markdown
如果一个动物长得像鸭子，行为像鸭子，那么它有极大的可能就是鸭子
```

鸭子类型是一种动态语言的风格，在这种风格下，一个对象的有效的语义，不是由继承自特定的类或者特定的接口，例如Java中的：

```java
public class Duck implements AnimalIface {...}
```

而是由**它当前的方法或者属性的集合**决定。Go作为一种现代静态语言，通过接口实现了鸭子类型，实际上是Go的编译器在其中做了隐匿转换。

##  2. <a name='-1'></a>探究的目标

- interface 如何构建，其内容如何组成。
- 动态分发是如何实现的，什么时候进行，并且有什么样的调用成本。
- 空接口和其它特殊情况有什么异同。
- 怎么组合 interface 完成工作。
- 如何进行断言，断言的成本有多高。

##  3. <a name='interface'></a>interface的组成部分

Go 语言根据接口类型是否包含一组方法将接口类型分成了两类：

- 使用 [`runtime.iface`](https://draveness.me/golang/tree/runtime.iface) 结构体表示包含方法的接口
- 使用 [`runtime.eface`](https://draveness.me/golang/tree/runtime.eface) 结构体表示不包含任何方法的 `interface{}` 类型；

[`runtime.eface`](https://draveness.me/golang/tree/runtime.eface) 结构体在 Go 语言中的定义是这样的：

```go
type eface struct { // 16 字节
	_type *_type
	data  unsafe.Pointer
}
```

由于 `interface{}` 类型不包含任何方法，所以它的结构也相对来说比较简单，只包含指向底层数据和类型的两个指针。从上述结构我们也能推断出 — Go 语言的任意类型都可以转换成 `interface{}`。

另一个用于表示接口的结构体是 [`runtime.iface`](https://draveness.me/golang/tree/runtime.iface)，这个结构体中有指向原始数据的指针 `data`，不过更重要的是 [`runtime.itab`](https://draveness.me/golang/tree/runtime.itab) 类型的 `tab` 字段。

```go
type iface struct { // 16 字节
	tab  *itab
	data unsafe.Pointer
}

type itab struct {
    inter *interfacetype
     _type *_type
    link *itab
    hash uint32 // copy of _type.hash. Used for type switches.
    bad bool // type does not implement interface
    inhash bool // has this itab been added to hash?
    unused [2]byte
    fun [1]uintptr // variable sized
}
```

`iface` 内部维护两个指针， `tab` 指向一个 `itab` 实体， 它表示接口的类型以及赋给这个 接口的实体类型。 `data` 则指向接口具体的值，一般而言是一个指向**堆内存**的指针。 再来仔细看一下 itab 结构体： `_type` 字段描述了实体的类型，包括内存对齐方式，大小 等； `inter` 字段则描述了接口的类型。

###  3.1. <a name='fun'></a>接口的方法数据——fun数组

为什么`fun`数组的大小为1，要是接口定义了多个方法怎么办？**实际上，这里存储的只是第一个方法的函数指针，如果有更多的方法，地址自增就行。另外，所有的方法都是按照方法名称的字典序进行排列的。**

 **`fun` 字段放置和接口方法对应的具体数据类型的方法地址，实现接口调用方法的动态分派，一般在每次给接口赋值发生转换时会更新此表，或者直接拿缓存的 `itab`**。 这里只会列出实体类型和接口相关的方法，实体类型的其他方法并不会出现在这里。如果你学过 C++ 的话，这里可以类比虚函数的概念。

###  3.2. <a name='interfacetype'></a>接口的类型描述——interfacetype

```go
type interfacetype struct {
	typ _type
	pkgpath name
	mhdr []imethod
}
```

可以看到，它包装了 `_type` 类型， `_type` 实际上是描述 Go 语言中各种数据类型的结构体。 我们注意到，这里还包含一个 `mhdr` 字段，表示接口所定义的函数列表， `pkgpath` 记录定义 了接口的包名。

这里通过一张图来看下 iface 结构体的全貌：

<div align=center><img src="/assets/if2.png"/></div>

###  3.3. <a name='_type'></a>类型的运行时表示—— _type

```go
type _type struct {
	size       uintptr
	ptrdata    uintptr
	hash       uint32
	tflag      tflag
	align      uint8
	fieldAlign uint8
	kind       uint8
	equal      func(unsafe.Pointer, unsafe.Pointer) bool
	gcdata     *byte
	str        nameOff
	ptrToThis  typeOff
}
```

- `size` 字段存储了类型占用的内存空间，为内存空间的分配提供信息；
- `hash` 字段能够帮助我们快速确定类型是否相等；
- `equal` 字段用于判断当前类型的多个对象是否相等，该字段是为了减少 Go 语言二进制包大小从 `typeAlg` 结构体中迁移过来的。

Go 语言各种数据类型都是在 `_type` 字段的基础上，增加一些额外的字段来进行管理的：

```go
type arraytype struct {
	typ _type
	elem *_type
	slice *_type
	len uintptr
}

type chantype struct {
	typ _type
	elem *_type
	dir uintptr
}

type slicetype struct {
	typ _type
	elem *_type
}

type structtype struct {
	typ _type
	pkgPath name
	fields []structfield
}
```

这些数据类型的结构体定义，是反射实现的基础。

##  4. <a name='-1'></a>值接收者和指针接收者

```go
type Math interface {
	Get() int
	Add()
}

type Adder struct {
	id int
}

func (adder Adder) Get() int {
	return adder.id
}

func (adder *Adder) Add() {
	adder.id++
}

func main() {
    // 值类型
	adderVal := Adder{id: 100}
    // 指针类型
	adderPtr := &Adder{id: 200}

	fmt.Println(adderVal.Get())
	adderVal.Add()
	fmt.Println(adderVal.Get())

	fmt.Println(adderPtr.Get())
	adderPtr.Add()
	fmt.Println(adderPtr.Get())
}
```

上述输出的结果为：

```go
PS D:\Github\repo\Go-Backend\src\g_interface> go run main.go
100
101
200
201
```

调用了`Add()`之后，无论是值接受者还是指针接受者的`id`都发生了改变，这实际上是编译器在背后做了一些工作：

|       -        |            值接收者             |                指针接收者                 |
| :------------: | :-----------------------------: | :---------------------------------------: |
|  **值调用者**  |   方法会使用调用者的一个副本    | 使用值的引用来调用方法`(&adderVal).Add()` |
| **指针调用者** | 指针被解引用`(*adderPtr).Add()` |        会使用调用者指针的一个副本         |

**实现了值接收者的方法，会隐含的实现了指针接收者的方法。**

###  4.1. <a name='-1'></a>两者分别在何时使用

如果方法的接收者是值类型，无论调用者是对象还是对象指针，修改的都是对象的副本，不影响调用 者；如果方法的接收者是指针类型，则调用者修改的是指针指向的对象本身。 使用指针作为方法的接收者的理由： 方法能够修改接收者指向的值。 避免在每次调用方法时复制该值，在值的类型为大型结构体时，这样做会更加高效。 **是使用值接收者还是指针接收者，不是由该方法是否修改了调用者（也就是接收者）来决定，而是应该基于该类型的本质** 。

如果类型具备“原始的本质”，也就是说**它的成员都是由 Go 语言里内置的原始类型，如字符串，整型值等，那就定义值接收者类型的方法** 。像**内置的引用类型，如 slice，map，interface， channel，这些类型比较特殊，声明他们的时候，实际上是创建了一个header ， 对于他们也是直接定义值接收者类型的方法**。这样，**调用函数时，是直接copy了这些类型的 header** ，而 header本身就是为复制设计的。 

如果类型具备非原始的本质，**不能被安全地复制**，这种类型总是应该被共享，那就定义指针接收者的方 法。比如 go源码里的文件结构体（struct File）就不应该被复制，应该只有一份 实体 。

##  5. <a name='nilinterface'></a>有关nil和interface

从`interface`源码可以看到，`iface`包含两个字段：

* `itab`是接口表指针，指向类型信息，**被称为动态类型**
* `data`是数据指针，指向具体的数据，**被称为动态值**

**而接口的零值是指动态类型和动态值都为nil，当且仅当这两部分的值都为nil的情况下，这个接口才会被认为是nil的** 。下面从一个例子来更深入了解这个特性：

```go
type Animal interface {
	Walk()
}

type Dog struct {
	name string
}

func (d *Dog) Walk() {
	fmt.Println("a dog is walking")
}

func InterfaceNil() {
	var animal Animal
	fmt.Println(animal == nil)
	fmt.Printf("%T %v\n", animal, animal)

	var dog *Dog
	fmt.Println(dog == nil)

	animal = dog
	fmt.Println(animal == nil)
	fmt.Printf("%T %v\n", animal, animal)
}

func main() {
	InterfaceNil()
}
```

程序的输出如下：

```shell
PS D:\Github\repo\Go-Backend\src\g_interface> go run main.go
true
<nil> <nil>
true
false
*main.Dog <nil>
```

* 一开始，`animal`的动态类型和静态类型都为`nil`，同时`dog`也为`nil`
* 当`animal`被赋值为`dog`的时候，`animal`的动态类型变成了`*Dog`，尽管此时动态值仍然为`nil`，但是动态类型不为`nil`，结果就不为`nil`

**因此，一个包含nil指针的接口不是nil接口**

##  6. <a name='interface-1'></a>interface的创建过程(太难了，慢慢写)

