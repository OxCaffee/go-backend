package main

import "fmt"

func main() {
	x, y := 1, 2
	var a = [2]*int{&x, &y}

	fmt.Println(*a[0])
}
