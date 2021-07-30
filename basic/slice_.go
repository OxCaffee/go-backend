package main

import "fmt"

// func makeSlice() {
// 	slice := make([]int, 0)
// 	fmt.Println(slice)
// }

// func appendSlice() {
// 	s := make([]int, 0)

// 	s = append(s, 1)

// 	fmt.Println(s)
// }

func cat() {
	s := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
	s1 := s[1:3]
	s2 := s1[2:7]

	fmt.Println(s)
	fmt.Println(s1)
	fmt.Println(s2)
}

func slmain() {
	// makeSlice()
	// appendSlice()
	cat()
}
