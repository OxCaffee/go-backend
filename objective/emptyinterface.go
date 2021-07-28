package main

import (
	"fmt"
)

type Person interface {
	Birth()
	Die()
	Toilet()
}

type Man struct {
	Name string
	Age  int
}

type Woman struct {
	Name string
	Age  int
}

func (man Man) Birth() {
	fmt.Println("A man named", man.Name, "borned")
}

func (man Man) Die() {
	fmt.Println("A man named", man.Name, "died")
}

func (man Man) Toilet() {
	fmt.Println("A man named", man.Name, "is going to the toilet")
}

func (woman *Woman) Birth() {
	fmt.Println("A woman named", woman.Name, "borned")
}

func (woman *Woman) Toilet() {
	fmt.Println("A woman named", woman.Name, "is going to the toilet")
}

func (woman *Woman) Die() {
	fmt.Println("A woman named", woman.Name, "died")
}

func emain() {
	persons := make([]Person, 0)

	persons = append(persons, Man{"lx", 100})
	persons = append(persons, &Woman{"lxx", 1000})

	for _, person := range persons {
		_, ok := person.(*Woman)

		if !ok {
			fmt.Println("This is a Man")
		} else {
			fmt.Println("This is a woman")
		}
	}
}
