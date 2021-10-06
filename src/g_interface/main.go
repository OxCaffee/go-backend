package main

import "fmt"

type Cat struct{}
type Duck interface {
	Walk()
	Yaya()
}

func (c Cat) Walk() {
	fmt.Println("this is cat walking")
}

func (c Cat) Yaya() {
	fmt.Println("this is cat yaya")
}

func ImplementInterfaceMethod() {
	var duck Duck = &Cat{}
	// fmt.Println(duck.Walk())
	duck.Walk()
}

func main() {
	ImplementInterfaceMethod()
}
