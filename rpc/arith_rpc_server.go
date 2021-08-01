package main

import (
	"errors"
	"google.golang.org/protobuf/proto"
	"net"
	"net/http"
	"net/rpc"
)

type A struct {
}

func (t *A) Add(req *ArithRequest, resp *ArithResponse) error {
	resp.C = proto.Int32(req.GetA() + req.GetB())
	return nil
}

func (t *A) Mul(req *ArithRequest, resp *ArithResponse) error {
	resp.C = proto.Int32(req.GetA() * req.GetB())
	return nil
}

func (t *A) Div(req *ArithRequest, resp *ArithResponse) error {
	if req.GetB() == 0 {
		return errors.New("divide by zero")
	}
	resp.C = proto.Int32(req.GetA() / req.GetB())
	return nil
}

func (t *A) Error(req *ArithRequest, resp *ArithResponse) error {
	return errors.New("erroroorororororor")
}

func AServer(){
	a := new(A)

	err := rpc.RegisterName("A", a)
	if err != nil {
		panic(err)
	}

	rpc.HandleHTTP()
	listen, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}
	_ = http.Serve(listen, nil)
}