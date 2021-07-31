package main

import "net/http"

func testWrite(w http.ResponseWriter, r *http.Request) {
	str := "this is a test for response writer\n"
	w.Write([]byte(str))
}

func main() {
	server := &http.Server{
		Addr: "127.0.0.1:8080",
	}

	http.HandleFunc("/write", testWrite)
	server.ListenAndServe()
}
