# unsafe.Pointer非安全类型指针

<!-- vscode-markdown-toc -->
* 1. [前言](#)
* 2. [unsafe.Pointer相关类型转换的编译规则](#unsafe.Pointer)
* 3. [使用unsafe.Pointer的基本运行时原则](#unsafe.Pointer-1)
* 4. [非类型安全指针相关的事实](#-1)
	* 4.1. [非类型安全指针值是指针但是uintptr值是整数](#uintptr)
	* 4.2. [不再被使用的内存块的回收时间是不确定的](#-1)
	* 4.3. [某些值的地址在程序运行期间可能改变](#-1)
* 5. [安全地使用非类型安全指针的使用模式](#-1)
	* 5.1. [将类型`*T1`的一个值转换为非类型安全指针值，然后将此非类型安全指针值转化为类型`*T2`](#T1T2)
	* 5.2. [将一个非类型安全指针值转换为一个uintptr值，然后使用此uintptr值](#uintptruintptr)
	* 5.3. [将一个非类型安全指针转换为一个uintptr值，然后此uintptr值参与各种算术运算，再将算数运算的结果uintptr值转换为非类型安全指针](#uintptruintptruintptr)
	* 5.4. [将reflect.Value.Pointer或者reflect.Value.UnsafeAddr方法的uintptr返回值转换为非类型安全指针](#reflect.Value.Pointerreflect.Value.UnsafeAddruintptr)
	* 5.5. [reflect.SliceHeader或者reflect.StringHeder值的Data字段和非类型安全指针之间的相互转换](#reflect.SliceHeaderreflect.StringHederData)
	* 5.6. [将非类型安全指针值转换为uintptr值并传递给syscall.Syscall函数调用](#uintptrsyscall.Syscall)

<!-- vscode-markdown-toc-config
	numbering=true
	autoSave=true
	/vscode-markdown-toc-config -->
<!-- /vscode-markdown-toc -->

##  1. <a name=''></a>前言

[`unsafe.Pointer`]() 指针类似于C语言中的无类型指针[`void*`]() ，借助[`unsafe.Pointer`]()有时候可以挽回Go运行时为了安全而牺牲的一些性能，因此必须小心按照官方文档中的使用说明使用[`unsafe.Pointer`]() 。稍有不慎，将会使Go类型系统(不包括非类型安全指针部分)精心设立内存安全壁垒的努力前功尽弃。同时，使用了[`unsafe.Pointer`]()的代码不受Go 1兼容性的保证。

##  2. <a name='unsafe.Pointer'></a>unsafe.Pointer相关类型转换的编译规则

* 一个类型安全指针值可以倍显式地转换为一个非类型安全指针类型
* 一个[`uintptr`]()值可以被显式地转换为一个非类型安全指针类型，反之亦然

**注意：这些规则是编译器接收的规则。满足这些规则的代码编译并没有什么问题，但是并不意味着在运行的时候是安全的。在使用非类型安全指针的时候，必须遵循一些原则以防止不安全的情况发生。**

##  3. <a name='unsafe.Pointer-1'></a>使用unsafe.Pointer的基本运行时原则

* 保证要使用的值在[`unsafe`]()操作前后要**时时刻刻被有效指针引用着，无论是类型安全指针还是非类型安全指针，否则此值有可能被GC掉**
* 任何指针都不应该引用未知内存块

##  4. <a name='-1'></a>非类型安全指针相关的事实

* 非类型安全指针是指针但是[`uintptr`]()是整数，整数从来不会引用其他的值
* 不再被使用的内存块的回收时间节点是不确定的
* 某些值的地址在程序运行中可能改变
* 一些值的声明范围可能并没有代码中看上去的大
* [`*unsafe.Pointer*`]()是一个类型安全的指针，它的基类型是[`unsafe.Pointer`]()

###  4.1. <a name='uintptr'></a>非类型安全指针值是指针但是uintptr值是整数

[`uintptr`]()值中时常用来存储内存地址的值，但是一个[`uintptr`]()值并不引用着存储于其中的地址处的值，所以此[`uintptr`]()值仍然在被使用但是无法防止存储与其中的地址处的值被回收。

**GC同等对待类型安全指针和非类型安全指针。只有指针可以引用其他值。**

###  4.2. <a name='-1'></a>不再被使用的内存块的回收时间是不确定的

启动一轮新的垃圾回收的过程的途径：

* GOGC环境变量，[`runtime/debug.SetGCPercent`]()
* 调用[`runtime.GC`]()函数来手动开启
* 最大垃圾回收时间间隔设置为分钟

###  4.3. <a name='-1'></a>某些值的地址在程序运行期间可能改变

为了提高程序的性能，每个协程维护一个栈（一段连续的内存块，64位系统上初始为2K）。在程序运行的时候，一个协程的栈的大小可能会根据需要而伸缩。当一个栈的大小改变的时候，[`runtime`]()会开辟出一段新的连续的内存块，并把老的连续的内存块的值复制到新的内存块上，从而相应的，开辟在此栈上的指针值中存储的地址可能发生改变。

**简而言之，目前开辟在栈上的值的地址可能发生改变，开辟在栈上的指针值中存储的值可能会发生改变。**

##  5. <a name='-1'></a>安全地使用非类型安全指针的使用模式

###  5.1. <a name='T1T2'></a>将类型`*T1`的一个值转换为非类型安全指针值，然后将此非类型安全指针值转化为类型`*T2`

**此操作的前提是T1的尺寸不小于T2** 。

```go
func ByteSlice2String(bs []byte) string {
	return *(*string)(unsafe.Pointer(&bs))
}
```

###  5.2. <a name='uintptruintptr'></a>将一个非类型安全指针值转换为一个uintptr值，然后使用此uintptr值

```go
func uintptrPtr() {
	type T struct{ a int }
	var t T
	fmt.Printf("%p\n", &t)
	println(&t)
	fmt.Printf("%x\n", uintptr(unsafe.Pointer(&t)))
}
```

输出结果

```go
0xc000014078
0xc000014078
c000014078
```

###  5.3. <a name='uintptruintptruintptr'></a>将一个非类型安全指针转换为一个uintptr值，然后此uintptr值参与各种算术运算，再将算数运算的结果uintptr值转换为非类型安全指针

```go
func unsafeCalculate() {
	type T struct {
		x bool
		y [3]int16
	}

	const N = unsafe.Offsetof(T{}.y)
	const M = unsafe.Sizeof(T{}.y[0])

	t := T{y: [3]int16{123, 456, 789}}
	p := unsafe.Pointer(&t)
	ty2 := (*int16)(unsafe.Pointer(uintptr(p) + N + M + M))	//  如果这里再加一个M，就出现错误了，因为越界了
	fmt.Println(*ty2)
}
```

输出结果：

```go
789
```

首先，[`T`]()结构体因为内存对齐的缘故，[`y`]()的偏移字节数为2，[`N`]()与[`M`]()都是2，结构体起始地址[`p`] + 2 + 2 + 2得到的就是[`T.y[2]`]()地址，也就是789

###  5.4. <a name='reflect.Value.Pointerreflect.Value.UnsafeAddruintptr'></a>将reflect.Value.Pointer或者reflect.Value.UnsafeAddr方法的uintptr返回值转换为非类型安全指针

设计的目的：避免不引用[`unsafe`]()包就可以将这两个方法的返回值(如果是[`unsafe.Pointer`]())类型转换为任何安全类型指针类型。**如果不立即转换为[`unsafe.Pointer`]()类型，将会出现一个可能导致处于返回的地址处的内存块被回收掉的时间窗口。**

```go
// 正确使用
p := (*int)(unsafe.Pointer(reflect.ValueOf(new(int)).Pointer()))

// 而下面的使用是危险的
u := reflect.ValueOf(new(int)).Pointer()
// 在这个时刻，处于存储在u中的地址处的内存块可能会被回收掉
p := (*int)(unsafe.Pointer(u))
```

###  5.5. <a name='reflect.SliceHeaderreflect.StringHederData'></a>reflect.SliceHeader或者reflect.StringHeder值的Data字段和非类型安全指针之间的相互转换

[`reflect.SliceHeader`]()和切片内部结构一致。[`reflect.StringHeader`]()和[`string`]()内部构造一致，**不要凭空生成[`StringHeader`]()或者[`SliceHeader`]()，应该从切片和字符串中去转换它们**

```go
func headerDataUnsafePtr() {
	bs := [...]byte{'n', 'i', 'h', 'a', 'o'}
	s := "golang"

	stringHeaderPtr := (*reflect.StringHeader)(unsafe.Pointer(&s))
	slicePtr := uintptr(unsafe.Pointer(&bs))
	stringHeaderPtr.Data = slicePtr
	stringHeaderPtr.Len = len(bs)

	fmt.Println(bs)
	fmt.Println(s)
}
```

输出结果

```go
[110 105 104 97 111]
nihao
```

可以看到，[`string`]()类型的字符串s底层的[`Data`]()指针被指向了[`[]byte`]()切片bs类型。

###  5.6. <a name='uintptrsyscall.Syscall'></a>将非类型安全指针值转换为uintptr值并传递给syscall.Syscall函数调用

```go
// 下面的操作是危险的
func func1(addr uintptr) {...}

// syscall标准库包调用是安全的
func Syscall(trap, a1, a2, a3 uintptr) (r1, r2 uintptr, err Errno)
```



