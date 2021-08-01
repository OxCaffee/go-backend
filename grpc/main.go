package main

import "time"

func main() {
	go Server()

	<-time.After(500 * time.Millisecond)

	Client()
}
