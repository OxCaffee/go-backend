package main

import (
	"fmt"
	"math"
)

func NanAsKey() {
	m := make(map[interface{}]interface{})
	m[math.NaN()] = 1
	m[math.NaN()] = 2

	for k, v := range m {
		fmt.Println(k, v)
	}
}

func main() {
	NanAsKey()
}