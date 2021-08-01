package main

import (
	"fmt"
	"net/rpc"
	"time"
)

func RpcClient() {
	client, err := rpc.Dial("tcp", ":8080")
	if err != nil {
		panic(err)
	}

	timeStamp := time.Now().Unix()
	req := OrderRequest{OrderId: "000001", TimeStamp: timeStamp}

	var  resp = new(OrderInfo)

	fmt.Println("call started")
	err = client.Call("S.GetOrderInfo", &req, resp)
	fmt.Println("call ended")
	if err != nil {
		panic(err)
	}

	fmt.Println(resp)
}