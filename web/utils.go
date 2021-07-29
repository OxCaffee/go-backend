package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"text/template"
	"web/data"
)

type Configuration struct {
	Address      string
	ReadTimeout  int64
	WriteTimeout int64
	Static       string
}

var config Configuration
var logger *log.Logger

// 标准输出
func p(a ...interface{}) {
	fmt.Println(a...)
}

func init() {
	loadConfig()
	file, err := os.OpenFile("./chichat.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)

	if err != nil {
		log.Fatalln("Failed to open log file", err)
	}

	logger = log.New(file, "INFO", log.Ldate|log.Ltime|log.Lshortfile)
}

func loadConfig() {
	file, err := os.Open("./config.json")

	if err != nil {
		log.Fatalln("Cannot open config file", err)
	}

	decoder := json.NewDecoder(file)
	config = Configuration{}
	err = decoder.Decode(&config)

	if err != nil {
		log.Fatalln("Cannot get configuration from file", err)
	}
}

func errorMessage(w http.ResponseWriter, req *http.Request, msg string) {
	url := []string{"/err?msg", msg}
	http.Redirect(w, req, strings.Join(url, ""), 302)
}

func session(w http.ResponseWriter, req *http.Request) (s data.Session, err error) {
	cookie, err := req.Cookie("_cookie")

	if err != nil {
		if cookie != nil {
			s = data.Session{Uuid: cookie.Value}
		}

		if ok, _ := s.Check(); !ok {
			err = errors.New("Invalid session")
		}
	}
	return
}

func parseTemplate(filenames ...string) (t *template.Template) {
	var files []string
	t = template.New("layout")

	for _, file := range filenames {
		files = append(files, fmt.Sprintf("template/%s.html", file))
	}

	t = template.Must(t.ParseFiles(files...))
	return
}

func generateHTML(w http.ResponseWriter, data interface{}, filenames ...string) {
	var files []string
	for _, file := range filenames {
		files = append(files, fmt.Sprintf("template/%s.html", file))
	}

	templates := template.Must(template.ParseFiles(files...))
	templates.ExecuteTemplate(w, "layout", data)
}

// for logging
func info(args ...interface{}) {
	logger.SetPrefix("INFO ")
	logger.Println(args...)
}

func danger(args ...interface{}) {
	logger.SetPrefix("ERROR ")
	logger.Println(args...)
}

func warning(args ...interface{}) {
	logger.SetPrefix("WARNING ")
	logger.Println(args...)
}

// version
func version() string {
	return "0.1"
}
