package main

import (
	"fmt"
	"reflect"
	"unsafe"
)

func Float64bits(f float64) uint64 {
	return *(*uint64)(unsafe.Pointer(&f))
}

func Float64Frombits(n uint64) float64 {
	return *(*float64)(unsafe.Pointer(&n))
}

func ByteSlice2String(bs []byte) string {
	return *(*string)(unsafe.Pointer(&bs))
}

func uintptrPtr() {
	type T struct{ a int }
	var t T
	fmt.Printf("%p\n", &t)
	println(&t)
	fmt.Printf("%x\n", uintptr(unsafe.Pointer(&t)))
}

func unsafeCalculate() {
	type T struct {
		x bool
		y [3]int16
	}

	const N = unsafe.Offsetof(T{}.y)
	const M = unsafe.Sizeof(T{}.y[0])

	t := T{y: [3]int16{123, 456, 789}}
	p := unsafe.Pointer(&t)
	ty2 := (*int16)(unsafe.Pointer(uintptr(p) + N + M + M))
	fmt.Println(*ty2)
}

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

func main() {
	// var f float64 = float64(2)
	// var i uint64 = uint64(2)

	// fmt.Println(Float64bits(f))
	// fmt.Println(Float64Frombits(i))

	// bs := []byte{'a', 'b', 'c'}
	// fmt.Println(ByteSlice2String(bs))
	// uintptrPtr()

	// unsafeCalculate()

	headerDataUnsafePtr()
}
