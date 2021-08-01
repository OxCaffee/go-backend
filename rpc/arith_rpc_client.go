package main

import (
	"fmt"
	"google.golang.org/protobuf/proto"
	"net/rpc"
)

func AClient() {
	req := new(ArithRequest)
	req.A = proto.Int32(123123)
	req.B = proto.Int32(23344)

	resp := new(ArithResponse)

	client, err := rpc.DialHTTP("tcp", ":8080")
	if err != nil {
		panic(err)
	}

	err = client.Call("A.Add", req, resp)
	if err != nil {
		panic(err)
	}

	fmt.Println(resp.C)
}