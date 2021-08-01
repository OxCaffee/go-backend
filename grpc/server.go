package main

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
)

const (
	port = ":8080"
)

type UserService struct {

}

func (userService *UserService) UserIndex(ctx context.Context, req *UserIndexRequest) (resp *UserIndexResponse, err error) {
	log.Printf("receive user index request: page %d page_size %d", req.Page, req.PageSize)

	return &UserIndexResponse{
		Err: 0,
		Msg: "success",
		Data: []*UserEntity{
			{Name: "Zhang San", Age: 28},
			{Name: "Li Si", Age: 32},
		},
	}, nil
}

func (userService *UserService) UserView(ctx context.Context, req *UserViewRequest) (resp *UserViewResponse, err error) {
	log.Printf("receive user view request: uid %d", req.Uid)

	return &UserViewResponse{
		Err:  0,
		Msg:  "success",
		Data: &UserEntity{Name: "james", Age: 28},
	}, nil
}

func (userService *UserService) UserPost(ctx context.Context, in *UserPostRequest) (*UserPostResponse, error) {
	log.Printf("receive user post request: name %s password %s age %d", in.Name, in.Password, in.Age)

	return &UserPostResponse{
		Err: 0,
		Msg: "success",
	}, nil
}

func (userService *UserService) UserDelete(ctx context.Context, in *UserDeleteRequest) (*UserDeleteResponse, error) {
	log.Printf("receive user delete request: uid %d", in.Uid)

	return &UserDeleteResponse{
		Err: 0,
		Msg: "success",
	}, nil
}

func Server() {
	listen, err := net.Listen("tcp", port)
	if err != nil {
		panic(err)
	}

	// 创建grpc服务器
	server := grpc.NewServer()
	// 为User实现注册业务，将User绑定到RPC服务器上
	RegisterUserServer(server, &UserService{})
	// 注册反射服务，这个服务是CLI使用的，根服务本身没有关系
	reflection.Register(server)

	if err := server.Serve(listen); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}