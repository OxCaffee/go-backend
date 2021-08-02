package main

import (
	"grpc/module"
	"time"
)

func main() {
	go Server()

	<-time.After(500 * time.Millisecond)

	Client()

	_ = module.M1{}
}
