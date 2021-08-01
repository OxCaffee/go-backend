package main

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"time"
)

const (
	addr = ":8080"
)

func Client() {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("didnot connect: %v", err)
	}

	defer func(conn *grpc.ClientConn) {
		err := conn.Close()
		if err != nil {
			log.Fatal("err closing connection")
		}
	}(conn)

	userClient := NewUserClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// UserIndex请求
	userIndexResponse, err := userClient.UserIndex(ctx, &UserIndexRequest{Page: 1, PageSize: 12})
	if err != nil {
		log.Fatal(err)
	}

	if 0 == userIndexResponse.Err {
		log.Printf("user index success: %s", userIndexResponse.Msg)
		// 包含UserEntity的数组
		userEntityList := userIndexResponse.Data
		for _, row := range userEntityList {
			fmt.Println(row.Name, row.Age)
		}
	} else {
		panic("userindex panic")
	}
}
