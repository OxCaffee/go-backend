package main

import (
	"fmt"
	"log"
	"net/rpc"
)

func SyncClient01() {
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

func AsyncClient01() {
	client, _ := rpc.DialHTTP("tcp", ":8080")
	args := &Args{7, 8}
	quotient := &Quotient{}
	asyncCall := client.Go("Arith.Divide", args, quotient, nil)
	replayCall := <-asyncCall.Done

	fmt.Println(replayCall.Reply)
}

func SyncClient02() {
	// 客户端
	client, err := rpc.DialHTTP("tcp", ":8080")
	if err != nil {
		panic(err)
	}

	// 参数和数据
	var req float32 = 1.1
	// 计算结果
	var reply float32

	err = client.Call("MathUtil.CalculateCircleArea", req, &reply)
	if err != nil {
		panic(err)
	}

	fmt.Println(reply)
}

func AsyncClient02() {
	// 客户端
	client, err := rpc.DialHTTP("tcp", ":8080")
	if err != nil {
		panic(err)
	}

	// 请求和结果
	var req float32 = 12.12
	var resp = new(float32)

	call := client.Go("MathUtil.CalculateCircleArea", req, resp, nil)
	done := <-call.Done
	fmt.Println(done)
	fmt.Println(*resp)
}
