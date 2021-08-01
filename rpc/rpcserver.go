package main

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
)

func Server01() {
	arith := new(Arith)

	_ = rpc.Register(arith)
	rpc.HandleHTTP()

	l, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal("listen error:", err)
	}

	_ = http.Serve(l, nil)
}

func Server02() {
	// 初始化指针类型
	mu := new(MathUtil)

	// 调用rpc的功能将服务进行注册
	err := rpc.Register(mu)
	if err != nil {
		panic(err)
	}

	// 注册到HTTP协议上
	rpc.HandleHTTP()

	// 设置监听
	listen, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}

	err = http.Serve(listen, nil)
	if err != nil {
		panic(err)
	}
}
