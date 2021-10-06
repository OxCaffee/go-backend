package main

//go:noinline
func add(a, b int) int {
	return a + b
}

func main() {
	add(1, 2)
}
