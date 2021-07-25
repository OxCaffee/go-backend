package main

import (
	"fmt"
)

// 大多数语言的递归版本，显式递归调用函数
func fact(n int) int {
	if n == 0 {
		return 1
	}
	return n * fact(n-1)
}

// go无显式递归版本fact(从函数的显式递归变成了函数的递归定义)
type DerecursiveFact func(DerecursiveFact, int) int

func factImpl(impl DerecursiveFact, n int) int {
	if n == 0 {
		return 1
	}
	return n * impl(impl, n-1)
}

func rmain() {
	fmt.Println(factImpl(factImpl, 10))
}
