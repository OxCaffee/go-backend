package main

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
)

func Server() {
	arith := new(Arith)

	_ = rpc.Register(arith)
	rpc.HandleHTTP()

	l, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal("listen error:", err)
	}

	_ = http.Serve(l, nil)
}
