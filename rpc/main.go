package main

import "time"

func main() {
	go Server()

	<-time.After(5 * time.Second)

	//Client()
	AsyncClient()
}