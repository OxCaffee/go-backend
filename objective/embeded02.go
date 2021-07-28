package main

import (
	"fmt"
	"reflect"
)

type A struct {
	fa1 string
	fa2 string
}

type B struct {
	*A
	fb1 string
	fb2 string
}

func (a A) Fam1()   { fmt.Println("fam1") }
func (a A) Fam2()   { fmt.Println("fam2") }
func (a *A) Fasm1() { fmt.Println("fasm1") }
func (a *A) Fasm2() { fmt.Println("fasm2") }
func (b B) Fbm1()   { fmt.Println("fbm1") }
func (b B) Fbm2()   { fmt.Println("fbm2") }
func (b *B) Fbsm1() { fmt.Println("fbsm1") }
func (b *B) Fbsm2() { fmt.Println("fbsm2") }

func main() {
	a := A{"as1", "as2"}
	b := B{&a, "bs1", "bs2"}

	fmt.Println("=========A struct==================")

	af := reflect.TypeOf(a)
	for i := 0; i < af.NumField(); i++ {
		fmt.Println("field", i, "in A is", af.Field(i))
	}

	for i := 0; i < af.NumMethod(); i++ {
		fmt.Println("method", i, "in A is", af.Method(i).Type)
	}

	afs := reflect.TypeOf(&a)
	for i := 0; i < afs.NumMethod(); i++ {
		fmt.Println("method", i, "in *A is", afs.Method(i))
	}

	fmt.Println("=========B struct==================")

	bf := reflect.TypeOf(b)
	for i := 0; i < bf.NumField(); i++ {
		fmt.Println("field", i, "in B is", bf.Field(i))
	}

	for i := 0; i < bf.NumMethod(); i++ {
		fmt.Println("method", i, "in B is", bf.Method(i))
	}

	bfs := reflect.TypeOf(&b)
	for i := 0; i < bfs.NumMethod(); i++ {
		fmt.Println("method", i, "in *B is", bfs.Method(i))
	}

	// 	=========A struct==================
	// field 0 in A is {fa1 main string  0 [0] false}
	// field 1 in A is {fa2 main string  16 [1] false}
	// method 0 in A is func(main.A)
	// method 1 in A is func(main.A)
	// method 0 in *A is {Fam1  func(*main.A) <func(*main.A) Value> 0}
	// method 1 in *A is {Fam2  func(*main.A) <func(*main.A) Value> 1}
	// method 2 in *A is {Fasm1  func(*main.A) <func(*main.A) Value> 2}
	// method 3 in *A is {Fasm2  func(*main.A) <func(*main.A) Value> 3}
	// =========B struct==================
	// field 0 in B is {A  *main.A  0 [0] true}
	// field 1 in B is {fb1 main string  8 [1] false}
	// field 2 in B is {fb2 main string  24 [2] false}
	// method 0 in B is {Fam1  func(main.B) <func(main.B) Value> 0}
	// method 1 in B is {Fam2  func(main.B) <func(main.B) Value> 1}
	// method 2 in B is {Fasm1  func(main.B) <func(main.B) Value> 2}
	// method 3 in B is {Fasm2  func(main.B) <func(main.B) Value> 3}
	// method 4 in B is {Fbm1  func(main.B) <func(main.B) Value> 4}
	// method 5 in B is {Fbm2  func(main.B) <func(main.B) Value> 5}
	// method 0 in *B is {Fam1  func(*main.B) <func(*main.B) Value> 0}
	// method 1 in *B is {Fam2  func(*main.B) <func(*main.B) Value> 1}
	// method 2 in *B is {Fasm1  func(*main.B) <func(*main.B) Value> 2}
	// method 3 in *B is {Fasm2  func(*main.B) <func(*main.B) Value> 3}
	// method 4 in *B is {Fbm1  func(*main.B) <func(*main.B) Value> 4}
	// method 5 in *B is {Fbm2  func(*main.B) <func(*main.B) Value> 5}
	// method 6 in *B is {Fbsm1  func(*main.B) <func(*main.B) Value> 6}
	// method 7 in *B is {Fbsm2  func(*main.B) <func(*main.B) Value> 7}
}
