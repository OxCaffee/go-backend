package main

import "fmt"

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
