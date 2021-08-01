package main

import (
	"fmt"
	"log"
	"net/rpc"
)

func SyncClient() {
	client, err := rpc.DialHTTP("tcp", ":8080")
	if err != nil {
		log.Fatal("dial rpc server failed")
	}

	args := &Args{7, 8}
	var reply int

	err2 := client.Call("Arith.Multiply", args, &reply)
	if err2 != nil {
		log.Fatal("error call rpc")
	}

	fmt.Printf("Arith: %d * %d = %d", args.A, args.B, reply)
}

func AsyncClient() {
	client, _ := rpc.DialHTTP("tcp", ":8080")
	args := &Args{7, 8}
	quotient := &Quotient{}
	asyncCall := client.Go("Arith.Divide", args, quotient, nil)
	replayCall := <-asyncCall.Done

	fmt.Println(replayCall.Reply)
}
