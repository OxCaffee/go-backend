package main

// type A struct {
// 	fas string
// 	fai int
// }

// func (a A) Amethod1() {
// 	fmt.Println("this is method 1 in A struct")
// }

// func (a A) Amethod2() {
// 	fmt.Println("this is method 2 in A struct")
// }

// type B struct {
// 	A
// 	fbs string
// 	fbi int
// }

// func (b *B) Bmethod1() {
// 	fmt.Println("this is method1 in *B struct")
// }

// func (b *B) Bmethod2() {
// 	fmt.Println("this is method2 in *B struct")
// }

// func main() {
// 	a := A{"a", 1}
// 	b := B{a, "b", 2}

// 	t := reflect.TypeOf(b)

// 	for i := 0; i < t.NumField(); i++ {
// 		fmt.Println("field", i, "in B is", t.Field(i))
// 	}

// 	for i := 0; i < t.NumMethod(); i++ {
// 		fmt.Println("method", i, "in B is", t.Method(i))
// 	}

// 	c := &b
// 	d := reflect.TypeOf(c)

// 	for i := 0; i < d.NumMethod(); i++ {
// 		fmt.Println("method", i, "in *B is", d.Method(i))
// 	}
// }
