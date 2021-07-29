package main

import (
	"fmt"
	"net/http"
	"web/data"
)

func newThread(w http.ResponseWriter, req *http.Request) {
	_, err := session(w, req)

	if err != nil {
		http.Redirect(w, req, "/login", 302)
	} else {
		generateHTML(w, nil, "layout", "private.navbar", "new.thread")
	}
}

func createThread(w http.ResponseWriter, req *http.Request) {
	sess, err := session(w, req)

	if err != nil {
		http.Redirect(w, req, "/login", 302)
	} else {
		err = req.ParseForm()

		if err != nil {
			danger(err, "Cannot parse form")
		}
		user, err := sess.User()

		if err != nil {
			danger(err, "Cannot get user from session")
		}
		topic := req.PostFormValue("topic")

		if _, err := user.CreateThread(topic); err != nil {
			danger(err, "Cannot create thread")
		}
		http.Redirect(w, req, "/", 302)
	}
}

func readThread(w http.ResponseWriter, req *http.Request) {
	vals := req.URL.Query()
	uuid := vals.Get("id")
	thread, err := data.ThreadByUUID(uuid)

	if err != nil {
		errorMessage(w, req, "Cannot read thread")
	} else {
		_, err := session(w, req)

		if err != nil {
			generateHTML(w, &thread, "layout", "public.navbar", "public.thread")
		} else {
			generateHTML(w, &thread, "layout", "private.navbar", "private.thread")
		}
	}
}

func postThread(writer http.ResponseWriter, request *http.Request) {
	sess, err := session(writer, request)
	if err != nil {
		http.Redirect(writer, request, "/login", 302)
	} else {
		err = request.ParseForm()
		if err != nil {
			danger(err, "Cannot parse form")
		}
		user, err := sess.User()
		if err != nil {
			danger(err, "Cannot get user from session")
		}
		body := request.PostFormValue("body")
		uuid := request.PostFormValue("uuid")
		thread, err := data.ThreadByUUID(uuid)
		if err != nil {
			errorMessage(writer, request, "Cannot read thread")
		}
		if _, err := user.CreatePost(thread, body); err != nil {
			danger(err, "Cannot create post")
		}
		url := fmt.Sprint("/thread/read?id=", uuid)
		http.Redirect(writer, request, url, 302)
	}
}
