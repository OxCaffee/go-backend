package main

import (
	"fmt"
	"reflect"
)

type Person struct {
	Name string
	Age  int
}

func (p Person) PrintName() {
	fmt.Println("Name:", p.Name)
}
func (p *Person) SetAge(age int) {
	p.Age = age
}

type Singer struct {
	Person // 通过内嵌Person类型来扩展之
	works  []string
}

func imain() {
	// var singer = Singer{Person: Person{"wwh", 123}}

	t := reflect.TypeOf(Singer{})
	fmt.Println("Singer有", t.NumField(), "个字段")
	fmt.Println("Singer有", t.NumMethod(), "个方法")

	for i := 0; i < t.NumField(); i++ {
		fmt.Println("Singer的第", i, "个字段为:", t.Field(i).Name)
	}

	for i := 0; i < t.NumMethod(); i++ {
		fmt.Println("Singer的第", i, "个方法为:", t.Method(i).Name)
	}

	p := reflect.TypeOf(Person{})
	fmt.Println("Person有", p.NumField(), "个字段")
	fmt.Println("Person有", p.NumMethod(), "个方法")

	for i := 0; i < p.NumField(); i++ {
		fmt.Println("Person的第", i, "个字段为:", p.Field(i).Name)
	}

	for i := 0; i < p.NumMethod(); i++ {
		fmt.Println("Person的第", i, "个方法为:", p.Method(i).Name)
	}

	tt := reflect.TypeOf(&Singer{})
	fmt.Println("*Singer有", tt.NumMethod(), "个方法")

	for i := 0; i < tt.NumMethod(); i++ {
		fmt.Println("*Singer的第", i, "个方法为:", tt.Method(i).Name)
	}
}
