package main

import (
	"net/http"
	"web/data"
)

// GET /login
// 展示登录界面
func login(w http.ResponseWriter, req *http.Request) {
	t := parseTemplate("lgoin.layout", "public.navbar", "login")
	t.Execute(w, nil)
}

// GET /signup
func signup(w http.ResponseWriter, req *http.Request) {
	generateHTML(w, nil, "login.layout", "public.navbar", "signup")
}

// POST /signup
func signupAccount(w http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()

	if err != nil {
		danger(err, "Cannot parse form")
	}
	user := data.User{
		Name:     req.PostFormValue("name"),
		Email:    req.PostFormValue("email"),
		Password: req.PostFormValue("password"),
	}

	if err := user.CreateUser(); err != nil {
		danger(err, "Cannot create user")
	}
	http.Redirect(w, req, "/login", 302)
}

func authenticate(w http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	user, err := data.UserByEmail(req.PostFormValue("email"))

	if err != nil {
		danger(err, "Cannot find user")
	}

	if user.Password == data.Encrypt(req.PostFormValue("password")) {
		session, err := user.CreateSession()

		if err != nil {
			danger(err, "Cannot create session")
		}

		cookie := http.Cookie{
			Name:     "_cookie",
			Value:    session.Uuid,
			HttpOnly: true,
		}

		http.SetCookie(w, &cookie)
		http.Redirect(w, req, "/", 302)
	} else {
		http.Redirect(w, req, "/login", 302)
	}
}

func logout(writer http.ResponseWriter, request *http.Request) {
	cookie, err := request.Cookie("_cookie")
	if err != http.ErrNoCookie {
		warning(err, "Failed to get cookie")
		session := data.Session{Uuid: cookie.Value}
		session.DeleteByUuid()
	}
	http.Redirect(writer, request, "/", 302)
}
