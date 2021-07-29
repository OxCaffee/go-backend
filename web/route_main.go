package main

import (
	"net/http"
	"web/data"
)

func err(w http.ResponseWriter, req *http.Request) {
	vals := req.URL.Query()
	_, err := session(w, req)

	if err != nil {
		generateHTML(w, vals.Get("msg"), "layout", "public.navbar", "error")
	} else {
		generateHTML(w, vals.Get("msg"), "layout", "private.navbar", "error")
	}
}

func index(w http.ResponseWriter, req *http.Request) {
	threads, err := data.Threads()

	if err != nil {
		errorMessage(w, req, "Cannot get threads")
	} else {
		_, err := session(w, req)

		if err != nil {
			generateHTML(w, threads, "layout", "public.navbar", "index")
		} else {
			generateHTML(w, threads, "layout", "private.navbar", "index")
		}
	}
}
