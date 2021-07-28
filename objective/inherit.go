package main

import (
	"fmt"
)

type Animal interface {
	Birth()
	Die()
	Live(food string) (energy int)
}

type Pig struct {
	Name string
	Age  int
}

func (pig Pig) Birth() {
	fmt.Println("A pig named", pig.Name, "borned..")
}

func (pig Pig) Die() {
	fmt.Println("A pig named", pig.Name, "died")
}

func (pig Pig) Live(food string) (energy int) {
	fmt.Println("A pig named", pig.Name, "is living and its age is", pig.Age)
	energy = 10000
	return
}

type Cat struct{}

func (cat *Cat) Birth() {
	fmt.Println("A cat borned")
}

func (cat *Cat) Die() {
	fmt.Println("A cat died")
}

func (cat *Cat) Live(food string) (energy int) {
	fmt.Println("A cat lived for 100 years")
	return 1000
}

func (cat *Cat) SayHello() {
	fmt.Println("A cat about", cat.Live("aaaa"), "saied hello")
}

func main01() {
	var pig, cat Animal
	pig = Pig{"lx", 100}
	cat = &Cat{}
	animals := make([]Animal, 0)
	animals = append(animals, pig)
	animals = append(animals, cat)

	for _, animal := range animals {
		animal.Birth()
		animal.Live("aaa")
		animal.Die()
	}

	fmt.Println()
}
