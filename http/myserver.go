package main

import (
	"fmt"
	"io"
	"net/http"
)

func handler1(w http.ResponseWriter, req *http.Request) {
	// 休眠2s

	// 验证csrf
	if req.Header.Get("_csrf") != "123456" {
		http.Error(w, "无效的Csrf!", 111)
		fmt.Println("无效的Csrf!")
	}

	// 验证Cookie
	if req.Header.Get("Cookie") != "abcd" {
		http.Error(w, "无效的Cookie", 112)
		fmt.Println("无效的Cookie")
	}

	io.WriteString(w, "这是处理器1\n")
}

func Server() {
	server := &http.Server{
		Addr: "127.0.0.1:8080",
	}

	http.HandleFunc("/handler1", handler1)

	server.ListenAndServe()
}
