package main

import "time"

func main() {
	go AServer()

	<-time.After(3*time.Second)

	AClient()
}