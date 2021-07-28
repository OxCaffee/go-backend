package main

import (
	"fmt"
	"reflect"
)

func main() {
	ro := reflect.TypeOf(1)
	io := reflect.ValueOf(ro)

	fmt.Println(ro, io)
}
