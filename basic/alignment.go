package main

import (
	"fmt"
	"unsafe"
)

func main01() {
	type T1 struct {
		a struct{}
		x int64
	}

	type T2 struct {
		x int64
		// 这部分需要有内存空间进行承载
		// 在32位系统中，64位数据实际上是由2个32位的数据存储
		// 因此x是4B+4B，此时a会对齐4B
		// 在64位系统中，64位数据由8B存储，因此a会填充8B，此时为16B
		a struct{}
	}

	fmt.Println("the size of T1 is: ", unsafe.Sizeof(T1{})) // 8
	fmt.Println("the size of T2 is: ", unsafe.Sizeof(T2{})) // 64位: 16, 32位: 12
	fmt.Println()
}

func main02() {
	type T struct {
		a int8       // start: 0, size: 1, padding: 7
		b int        // start: 8, size: 8, padding: 0
		c int16      // start: 16,size: 2, padding: 2
		d int32      // start: 20,size: 4, padding: 0
		e int64      // start: 24,size: 8, padding: 0
		f complex128 // start: 32,size: 8, padding:24
		g [2]string  // start: 48,size: 32,padding: 0
		h bool       // start: 80,size: 1, padding: 0
		i struct{}   // start: 81,size: 0, padding: 7 <=====
		j [0]int     // start: 88,size: 0, padding: 8 <=====思考????????????????
	}

	fmt.Println("the size of struct T is:", unsafe.Sizeof(T{}))
	fmt.Println("s:", unsafe.Offsetof(T{}.a), "; size:", unsafe.Sizeof(T{}.a))
	fmt.Println("s:", unsafe.Offsetof(T{}.b), "; size:", unsafe.Sizeof(T{}.b))
	fmt.Println("s:", unsafe.Offsetof(T{}.c), "; size:", unsafe.Sizeof(T{}.c))
	fmt.Println("s:", unsafe.Offsetof(T{}.d), "; size:", unsafe.Sizeof(T{}.d))
	fmt.Println("s:", unsafe.Offsetof(T{}.e), "; size:", unsafe.Sizeof(T{}.e))
	fmt.Println("s:", unsafe.Offsetof(T{}.f), "; size:", unsafe.Sizeof(T{}.f))
	fmt.Println("s:", unsafe.Offsetof(T{}.g), "; size:", unsafe.Sizeof(T{}.g))
	fmt.Println("s:", unsafe.Offsetof(T{}.h), "; size:", unsafe.Sizeof(T{}.h))
	fmt.Println("s:", unsafe.Offsetof(T{}.i), "; size:", unsafe.Sizeof(T{}.i))
	fmt.Println("s:", unsafe.Offsetof(T{}.j), "; size:", unsafe.Sizeof(T{}.j))
}

func main03() {
	type T struct {
		a int8     // start: 0, size: 1, padding: 0
		b struct{} // start: 1, size: 0, padding: 7
		c [0]int   // start: 8, size: 0, padding: 0
		d struct{} // start: 8, size: 0, padding: 8
		e [0]int   // start: 8, size: 0, padding: 8
	}

	fmt.Println("the size of struct T is:", unsafe.Sizeof(T{}))
	fmt.Println("s:", unsafe.Offsetof(T{}.a), "; size:", unsafe.Sizeof(T{}.a))
	fmt.Println("s:", unsafe.Offsetof(T{}.b), "; size:", unsafe.Sizeof(T{}.b))
	fmt.Println("s:", unsafe.Offsetof(T{}.c), "; size:", unsafe.Sizeof(T{}.c))
	fmt.Println("s:", unsafe.Offsetof(T{}.d), "; size:", unsafe.Sizeof(T{}.d))
	fmt.Println("s:", unsafe.Offsetof(T{}.e), "; size:", unsafe.Sizeof(T{}.e))
}

func main04() {
	type T struct {
		a struct{} // start: 0, size: 0, padding: 0
		b [0]int   // start: 0, size: 0, padding: 0
	}

	fmt.Println("the size of struct T is:", unsafe.Sizeof(T{}))
	fmt.Println("s:", unsafe.Offsetof(T{}.a), "; size:", unsafe.Sizeof(T{}.a))
	fmt.Println("s:", unsafe.Offsetof(T{}.b), "; size:", unsafe.Sizeof(T{}.b))
}

func main() {
	type T struct {
		a int8
		b struct{}
		c [0]int
	}
	fmt.Println("the size of struct T is:", unsafe.Sizeof(T{}))
	fmt.Println("s:", unsafe.Offsetof(T{}.a), "; size:", unsafe.Sizeof(T{}.a))
	fmt.Println("s:", unsafe.Offsetof(T{}.b), "; size:", unsafe.Sizeof(T{}.b))
	fmt.Println("s:", unsafe.Offsetof(T{}.c), "; size:", unsafe.Sizeof(T{}.c))
}
