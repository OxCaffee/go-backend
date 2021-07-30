# slice切片源码解读

<!-- vscode-markdown-toc -->
* 1. [前言](#)
* 2. [slice结构体定义](#slice)
* 3. [slice初始化](#slice-1)
	* 3.1. [slice计算所需内存大小](#slice-1)
	* 3.2. [slice分配内存——源码太长太过底层以后再补](#slice-1)
* 4. [slice的容量增长](#slice-1)
* 5. [slice的截取](#slice-1)
* 6. [slice深拷贝](#slice-1)
* 7. [slice值传递还是引用传递](#slice-1)

<!-- vscode-markdown-toc-config
	numbering=true
	autoSave=true
	/vscode-markdown-toc-config -->
<!-- /vscode-markdown-toc -->

##  1. <a name=''></a>前言

[`slice`]()是数组在Golang中的动态表现形式，即描述的是动态的数组，可以向[`slice`]()后追加元素，在[`slice`]()容量不足的时候还可以自动地进行扩充。

##  2. <a name='slice'></a>slice结构体定义

[`slice`]()结构体的定义如下：

```go
// src/runtime/slice.go

type slice struct {
	array unsafe.Pointer	// unsafe.Pointer 本质上还是一个int型的指针
	len   int				// 当前有效数组的长度
	cap   int				// 当前数组的容量，len>cap时会发生扩容操作
}
```

在`slice.go`文件中，还定义了一种**不在堆heap中分配内存的slice结构** ，即[`notInHeapSlice`]() ：

```go
// src/runtime/slice.go

type notInHeapSlice struct {
    array *notInHeap	// type notInHeap struct{}
	len   int			// 长度
	cap   int			// 容量
}
```

##  3. <a name='slice-1'></a>slice初始化

###  3.1. <a name='slice-1'></a>slice计算所需内存大小

首先先来看一段代码：

```go
import "fmt"

func makeSlice() {
	slice := make([]int, 0)
	fmt.Println(slice)
}

func main() {
	makeSlice()
}
```

上面的代码创建了一个[`int`]()类型的[`slice`]()切片，利用[`go tool compile -S slice.go`]()查看汇编代码得到如下结果：

```assembly
"".makeSlice STEXT size=202 args=0x0 locals=0x58 funcid=0x0
        0x0000 00000 (.\slice_.go:5)    TEXT    "".makeSlice(SB), ABIInternal, $88-0
		# 此处省略若干行
		# 调用了runtime.makeslice创建切片数组
        0x0040 00064 (.\slice_.go:6)    CALL    runtime.makeslice(SB)
		# 此处省略若干行
		# 调用了runtime.convTslice
        0x0056 00086 (.\slice_.go:7)    CALL    runtime.convTslice(SB)
		# 此处省略若干行
		# 调用runtime.morestack_noctxt
        0x00c0 00192 (.\slice_.go:5)    CALL    runtime.morestack_noctxt(SB)
		# 之后的汇编内容省略
```

可以看到[`slice`]()的创建实际上调用了[`runtime`]()包下的[`makeslice`]()方法，根据汇编代码的指示我们查看[`makeslice`]()的源码，如下：

```go
// src/runtime/slice.go

func makeslice(et *_type, len, cap int) unsafe.Pointer {
    // <1> 计算所需内存
	mem, overflow := math.MulUintptr(et.size, uintptr(cap))
	if overflow || mem > maxAlloc || len < 0 || len > cap {
		mem, overflow := math.MulUintptr(et.size, uintptr(len))
		if overflow || mem > maxAlloc || len < 0 {
			panicmakeslicelen()
		}
		panicmakeslicecap()
	}

    // 根据计算所需内存分配内存
	return mallocgc(mem, et, true)
}
```

可以看到[`makeslice`]()内部调用了[`MulUintptr`]()来**决定分配的mem空间大小以及是否溢出overlow的信息** ，[`math.MulUintptr`]()方法源码如下：

```go
func MulUintptr(a, b uintptr) (uintptr, bool) {
	if a|b < 1<<(4*sys.PtrSize) || a == 0 {
		return a * b, false
	}
	overflow := b > MaxUintptr/a
	return a * b, overflow
}
```

[`math.MulUintptr`]()用切片元素大小和切片容量相乘计算出所需占用的内存空间，如果内存溢出或者计算出的内存大小大于最大可分配内存，[`overflow`]()就会返回[`true`]() ，同时[`makeslice`]()报错。

我们继续回到[`makeslice`]()方法中，可以看到，为[`slice`]()计算分配内存大小实际上执行了两次：

* 一次是根据[`capacity`]()最大容量尝试计算，如果没有出现错误则执行分配内存
* 如果第一次计算不成功，则根据传参中的[`length`]()长度计算分配内存，没有出现错误则执行分配内存，如果出现错误，不分配内存，返回错误

###  3.2. <a name='slice-1'></a>slice分配内存——源码太长太过底层以后再补

[`slice`]()分配内存调用的是[`mallocgc`]()方法。

##  4. <a name='slice-1'></a>slice的容量增长

下面我们演示了[`append`]()操作触发[`slice`]()容量扩充，先上示例代码：

```go
import "fmt"

func appendSlice() {
	s := make([]int, 0)
	s = append(s, 1)
	fmt.Println(s)
}

func main() {
	appendSlice()
}
```

调用[`go tool compile -S slice_.go`]()查看并截取汇编代码如下：

```assembly
"".appendSlice STEXT size=266 args=0x0 locals=0x58 funcid=0x0
        0x0000 00000 (.\slice_.go:10)   TEXT    "".appendSlice(SB), ABIInternal, $88-0
		# 首先先makeslice，初始len=0
        0x0040 00064 (.\slice_.go:11)   CALL    runtime.makeslice(SB)
		# 执行append之后，调用了growslice扩充slice容量
        0x006b 00107 (.\slice_.go:13)   CALL    runtime.growslice(SB)
    	# 打印
        0x00ee 00238 ($GOROOT\src\fmt\print.go:274)     CALL    fmt.Fprintln(SB)
		# context相关
        0x0100 00256 (.\slice_.go:10)   CALL    runtime.morestack_noctxt(SB)
```

可以看到，与[`slice`]()容量有关的增长调用的是[`runtime.growslice`]()操作，源码如下：

```go
// src/runtime/slice.go

func growslice(et *_type, old slice, cap int) slice {
    // <1> 与old切片相关的操作
    if raceenabled {
		callerpc := getcallerpc()
		racereadrangepc(old.array, uintptr(old.len*int(et.size)), callerpc, funcPC(growslice))
	}
	if msanenabled {
		msanread(old.array, uintptr(old.len*int(et.size)))
	}
    
    // <2> 处理新分配的slice容量溢出问题
    // 首先，new_cap一定比old_cap大，但是如果new_cap<old_cap，则说明输出上溢，超过了分配的空间大小
	if cap < old.cap {
		panic(errorString("growslice: cap out of range"))
	}
    
    // <3> 处理空指针的问题，append操作不能产生一个长度不为0但是指针为0的slice
    // 因此默认情况下，slice新创建的指针指向空间的默认值
    // 这里使用到了zerobase, zerobase是所有0字节指针的基地址
	if et.size == 0 {
		return slice{unsafe.Pointer(&zerobase), old.len, cap}
	}
    
    // <4> 开始决定扩充后的容量
	newcap := old.cap	// 为什么先保存这一步？因为新的cap可能溢出，所以需要保存old cap
	doublecap := newcap + newcap	// 尝试cap翻倍扩充
    if cap > doublecap {
		newcap = cap	// 溢出了，新的newcap只能是原来的cap
	} else {
		if old.cap < 1024 {	// 如果原来的cap小于1024，newcap为oldcap的2倍
			newcap = doublecap
		} else {	// 如果old cap >= 1024，检查newcap是否溢出
            
			for 0 < newcap && newcap < cap {
				// 注意这里cap增加为原来的1.25倍
                newcap += newcap / 4 
			}
			// 如果newcap溢出，继续使用原来的cap
			if newcap <= 0 {
				newcap = cap
			}
		}
	}
    
    // <5> 根据传入类型参数的大小决定内存分配的一些参数lenmem,newlenmem和capmem等等
  	// 这里用到了内存对齐，也就是说，正常情况下是1.5和1.25倍地扩充，但是当需要内存对齐地时候，可能就会多扩充一些
    // ...
    
    // <6> 获得新的slice内存相关数据，例如指针等等，这里执行了内存的分配，其中新的slice中的array指针为p
    var p unsafe.Pointer
	if et.ptrdata == 0 {
		p = mallocgc(capmem, nil, false)
		// The append() that calls growslice is going to overwrite from old.len to cap (which will be the new length).
		// Only clear the part that will not be overwritten.
		memclrNoHeapPointers(add(p, newlenmem), capmem-newlenmem)
	} else {
		// Note: can't use rawmem (which avoids zeroing of memory), because then GC can scan uninitialized memory.
		p = mallocgc(capmem, et, true)
		if lenmem > 0 && writeBarrier.enabled {
			// Only shade the pointers in old.array since we know the destination slice p
			// only contains nil pointers because it has been cleared during alloc.
			bulkBarrierPreWriteSrcOnly(uintptr(p), uintptr(old.array), lenmem-et.size+et.ptrdata)
		}
	}
    
    // <7> 将old slice数据复制到新的new slice中
    memmove(p, old.array, lenmem)
    
    // <8> 返回创建后的新的slice
    return slice{p, old.len, newcap}
}
```

**从上面可以得到[`slice`]()的简单扩容机制，即原来[`slice`]()的cap小于1024的时候，新的[`slice`]()变成原来的2倍，反之变成1.25倍**

**但是上面只是简单的扩容规则，实际上因为内存对齐的一些原因，可能实际扩充的容量要稍大一些**

##  5. <a name='slice-1'></a>slice的截取

**slice的截取的使用一定要非常小心，因为很容易出现bug** 。下面先来看一段代码：

```go
package main

import "fmt"

func main() {
      slice := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
      s1 := slice[2:5]
      s2 := s1[2:7]
      fmt.Printf("len=%-4d cap=%-4d slice=%-1v n", len(slice), cap(slice), slice)
      fmt.Printf("len=%-4d cap=%-4d s1=%-1v n", len(s1), cap(s1), s1)
      fmt.Printf("len=%-4d cap=%-4d s2=%-1v n", len(s2), cap(s2), s2)
}

```

程序输出：

```go
len=10   cap=10   slice=[0 1 2 3 4 5 6 7 8 9] 
len=3    cap=8    s1=[2 3 4] 
len=5    cap=6    s2=[4 5 6 7 8]
```

s1的长度变成3，cap变为8（默认截取到最大容量）， 但是s2截取s1的第2到第7个元素，左闭右开，很多人想问，s1根本没有那么元素啊，但是实际情况是s2截取到了，并且没有发生数组越界，原因就是s2实际截取的是底层数组，**目前slice、s1、s2都是共用的同一个底层数组 ** 。

我们继续操作：

```go
fmt.Println("--------append 100----------------")
s2 = append(s2, 100)
```

输出结果是：

```go
--------append 100----------------
len=10   cap=10   slice=[0 1 2 3 4 5 6 7 8 100] 
len=3    cap=8    s1=[2 3 4] 
len=6    cap=6    s2=[4 5 6 7 8 100]
```

我们看到往s2里append数据影响到了slice，正是因为两者底层数组是一样的；但是既然都是共用的同一底层数组，s1为什么没有100，这个问题再下一节会讲到，大家稍安勿躁。我们继续进行操作：

```go
fmt.Println("--------append 200----------------")
s2 = append(s2, 200)
```

输出结果是：

```go
--------append 200----------------
len=10   cap=10   slice=[0 1 2 3 4 5 6 7 8 100] 
len=3    cap=8    s1=[2 3 4] 
len=7    cap=12   s2=[4 5 6 7 8 100 200]
```

我们看到继续往s2中append一个200，但是只有s2发生了变化，slice并未改变，为什么呢？对，是因为在append完100后，s2的容量已满，再往s2中append，底层数组发生复制，系统分配了一块新的内存地址给s2，s2的容量也翻倍了。

我们继续操作：

```go
fmt.Println("--------modify s1----------------")
s1[2] = 20
```

输出会是什么样呢？

```go
--------modify s1----------------
len=10   cap=10   slice=[0 1 2 3 20 5 6 7 8 100] 
len=3    cap=8    s1=[2 3 20] 
len=7    cap=12   s2=[4 5 6 7 8 100 200]
```

这就很容易理解了，我们对s1进行更新，影响了slice，因为两者共用的还是同一底层数组，s2未发生改变是因为在上一步时底层数组已经发生了变化；

以此来看，slice截取的坑确实很多，极容易出现bug，并且难以排查，大家在使用的时候一定注意。

##  6. <a name='slice-1'></a>slice深拷贝

上一节中对slice进行的截取，新的slice和原始slice共用同一个底层数组，因此可以看做是对slice的浅拷贝，那么在go中如何实现对slice的深拷贝呢？那么就要依赖golang提供的copy函数了，我们用一段程序来简单看下如何实现深拷贝：

```go
func main() {

  // Creating slices
  slice1 := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
  var slice2 []int
  slice3 := make([]int, 5)

  // Before copying
  fmt.Println("------------before copy-------------")
  fmt.Printf("len=%-4d cap=%-4d slice1=%vn", len(slice1), cap(slice1), slice1)
  fmt.Printf("len=%-4d cap=%-4d slice2=%vn", len(slice2), cap(slice2), slice2)
  fmt.Printf("len=%-4d cap=%-4d slice3=%vn", len(slice3), cap(slice3), slice3)


  // Copying the slices
  copy_1 := copy(slice2, slice1)
  fmt.Println()
  fmt.Printf("len=%-4d cap=%-4d slice1=%vn", len(slice1), cap(slice1), slice1)
  fmt.Printf("len=%-4d cap=%-4d slice2=%vn", len(slice2), cap(slice2), slice2)
  fmt.Println("Total number of elements copied:", copy_1)
}
```

首先定义了三个slice，然后将slice1 copy到slice2，我们来看下输出结果：

```go
------------before copy-------------
len=10   cap=10   slice1=[0 1 2 3 4 5 6 7 8 9]
len=0    cap=0    slice2=[]
len=5    cap=5    slice3=[0 0 0 0 0]

len=10   cap=10   slice1=[0 1 2 3 4 5 6 7 8 9]
len=0    cap=0    slice2=[]
Total number of elements copied: 0
```

我们发现slice1的内容并未copy到slice2，为什么呢？我们再试下将slice1 copy到slice3，如下：

```go
copy_2 := copy(slice3, slice1)
```

输出结果：

```go
len=10   cap=10   slice1=[0 1 2 3 4 5 6 7 8 9]
len=5    cap=5    slice3=[0 1 2 3 4]
Total number of elements copied: 5
```

我们看到copy成功，slice3和slice2唯一的区别就是slice3的容量为5，而slice2容量为0，那么是否是深拷贝呢，我们修改slice3的内容看下：

```go
slice3[0] = 100
```

我们再看下输出结果：

```go
len=10   cap=10   slice1=[0 1 2 3 4 5 6 7 8 9]
len=5    cap=5    slice3=[100 1 2 3 4]
```

我们可以看到修改slice3后，slice1的值并未改变，可见copy实现的是深拷贝。由此可见，copy函数为slice提供了深拷贝能力，但是需要在拷贝前申请内存空间。参照makeslice和growslice我们对本节一开始的程序进行反汇编，得到汇编代码（部分）如下：

```go
0x0080 00128 (slice.go:10)  CALL  runtime.makeslice(SB)
  0x0085 00133 (slice.go:10)  PCDATA  $0, $1
  0x0085 00133 (slice.go:10)  MOVQ  24(SP), AX
  0x008a 00138 (slice.go:10)  PCDATA  $1, $2
  0x008a 00138 (slice.go:10)  MOVQ  AX, ""..autotmp_75+96(SP)
  0x008f 00143 (slice.go:11)  PCDATA  $0, $4
  0x008f 00143 (slice.go:11)  MOVQ  ""..autotmp_74+104(SP), CX
  0x0094 00148 (slice.go:11)  CMPQ  AX, CX
  0x0097 00151 (slice.go:11)  JEQ  176
  0x0099 00153 (slice.go:11)  PCDATA  $0, $5
  0x0099 00153 (slice.go:11)  MOVQ  AX, (SP)
  0x009d 00157 (slice.go:11)  PCDATA  $0, $0
  0x009d 00157 (slice.go:11)  MOVQ  CX, 8(SP)
  0x00a2 00162 (slice.go:11)  MOVQ  $40, 16(SP)
  0x00ab 00171 (slice.go:11)  CALL  runtime.memmove(SB)
  0x00b0 00176 (slice.go:12)  MOVQ  $10, (SP)
  0x00b8 00184 (slice.go:12)  CALL  runtime.convT64(SB)
```

我们发现copy函数其实是调用runtime.memmove，其实我们在研究runtime/slice.go文件中的源码的时候，会发现有一个slicecopy函数，这个函数最终就是调用runtime.memmove来实现slice的copy的，我们看下源码：

```go
func slicecopy(to, fm slice, width uintptr) int {
  // 如果源切片或者目标切片有一个长度为0，那么就不需要拷贝，直接 return 
  if fm.len == 0 || to.len == 0 {
    return 0
  }

  // n 记录下源切片或者目标切片较短的那一个的长度
  n := fm.len
  if to.len < n {
    n = to.len
  }

  // 如果入参 width = 0，也不需要拷贝了，返回较短的切片的长度
  if width == 0 {
    return n
  }

  //如果开启竞争检测
  if raceenabled {
    callerpc := getcallerpc()
    pc := funcPC(slicecopy)
    racewriterangepc(to.array, uintptr(n*int(width)), callerpc, pc)
    racereadrangepc(fm.array, uintptr(n*int(width)), callerpc, pc)
  }
  if msanenabled {
    msanwrite(to.array, uintptr(n*int(width)))
    msanread(fm.array, uintptr(n*int(width)))
  }

  size := uintptr(n) * width
  if size == 1 { // common case worth about 2x to do here
    // TODO: is this still worth it with new memmove impl?
    //如果只有一个元素，那么直接进行地址转换
    *(*byte)(to.array) = *(*byte)(fm.array) // known to be a byte pointer
  } else {
    //如果不止一个元素，那么就从 fm.array 地址开始，拷贝到 to.array 地址之后，拷贝个数为size
    memmove(to.array, fm.array, size)
  }
  return n
}
```

##  7. <a name='slice-1'></a>slice值传递还是引用传递

slice在作为函数参数进行传递的时候，是值传递还是引用传递，我们来看一段程序：

```go
package main

import "fmt"

func main() {
  slice := make([]int, 0, 10)
  slice = append(slice, 1)
  fmt.Println(slice, len(slice), cap(slice))
  fn(slice)
  fmt.Println(slice, len(slice), cap(slice))
}
func fn(in []int) {
  in = append(in, 5)
}
```

很简单的一段程序，我们直接来看输出结果：

```go
[1] 1 10
[1] 1 10
```

可见fn内的append操作并未对slice产生影响，那我们再看一段代码：

```go
package main

import "fmt"

func main() {
  slice := make([]int, 0, 10)
  slice = append(slice, 1)
  fmt.Println(slice, len(slice), cap(slice))
  fn(slice)
  fmt.Println(slice, len(slice), cap(slice))
}
func fn(in []int) {
  in[0] = 100
}
```

输出是什么？我们来看下：

```go
[1] 1 10
[100] 1 10
```

slice居然改变了，是不是有点混乱？前面我们说到slice底层其实是一个结构体，len、cap、array分别表示长度、容量、底层数组的地址，当slice作为函数的参数传递的时候，跟普通结构体的传递是没有区别的；如果直接传slice，实参slice是不会被函数中的操作改变的，但是如果传递的是slice的指针，是会改变原来的slice的；另外，无论是传递slice还是slice的指针，如果改变了slice的底层数组，那么都是会影响slice的，这种通过数组下标的方式更新slice数据，是会对底层数组进行改变的，所以就会影响slice。

那么，讲到这里，在第一段程序中在fn函数内append的5到哪里去了，不可能凭空消失啊，我们再来看一段程序：

```go
package main

import "fmt"

func main() {
  slice := make([]int, 0, 10)
  slice = append(slice, 1)
  fmt.Println(slice, len(slice), cap(slice))
  fn(slice)
  fmt.Println(slice, len(slice), cap(slice))
  s1 := slice[0:9]//数组截取
  fmt.Println(s1, len(s1), cap(s1))
}
func fn(in []int) {
  in = append(in, 5)
}
```

我们来看输出结果：

```go
[1] 1 10
[1] 1 10
[1 5 0 0 0 0 0 0 0] 9 10
```

显然，虽然在append后，slice中并未展示出5，也无法通过slice[1]取到（会数组越界）,但是实际上底层数组已经有了5这个元素，但是由于slice的len未发生改变，所以我们在上层是无法获取到5这个元素的。那么，再问一个问题，我们是不是可以手动强制改变slice的len长度，让我们可以获取到5这个元素呢？是可以的，我们来看一段程序：

```go
package main

import (
  "fmt"
  "reflect"
  "unsafe"
)

func main() {
  slice := make([]int, 0, 10)
  slice = append(slice, 1)
  fmt.Println(slice, len(slice), cap(slice))
  fn(slice)
  fmt.Println(slice, len(slice), cap(slice))
  (*reflect.SliceHeader)(unsafe.Pointer(&slice)).Len = 2 //强制修改slice长度
  fmt.Println(slice, len(slice), cap(slice))
}

func fn(in []int) {
  in = append(in, 5)
}
```

我们来看输出结果：

```go
[1] 1 10
[1] 1 10
[1 5] 2 10
```

可以看出，通过强制修改slice的len，我们可以获取到了5这个元素。

所以再次回答一开始我们提出的问题，slice是值传递还是引用传递？答案是值传递！

以上，在使用golang中的slice的时候大家一定注意，否则稍有不慎就会出现bug。