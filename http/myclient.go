package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

var client = http.Client{}

// 发送get请求
func Get(url string) (resp *http.Response, err error) {
	req, err := http.NewRequest("GET", url, nil)

	// 设置头部
	req.Header.Set("_csrf", "123456")
	req.Header.Set("Cookie", "abcd")

	if err != nil {
		return nil, err
	}

	return client.Do(req)
}

func handleResponse(resp *http.Response) {
	fmt.Printf("resp.Header: %v\n", resp.Header)

	bytes, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		fmt.Println("响应解析错误!")
	}

	fmt.Println("响应信息: ", bytes)
}

func mmmmmain() {

	url := "http://localhost:8080/handler1"

	r, err := Get(url)

	if err != nil {
		fmt.Println("请求出现错误！")
	} else {
		handleResponse(r)
	}
}
