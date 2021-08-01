package main

import "net/http"

func testWrite(w http.ResponseWriter, r *http.Request) {
	str := "this is a test for response writer\n"
	w.Write([]byte(str))
}

func addTrailers() {
	http.HandleFunc("/addTrailers", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Trailer", "t1, t2")
		w.Header().Add("Trailer", "t3")

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("t1", "trailer1")
		w.Header().Set("t2", "trailer2")
		w.Write([]byte("trailers"))
	})
}

func main() {
	server := &http.Server{
		Addr: "127.0.0.1:8080",
	}

	http.HandleFunc("/write", testWrite)
	addTrailers()
	server.ListenAndServe()
}
