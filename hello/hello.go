package hello

import (
	"rsc.io/quote"
)

func Hello() string {
	return "hello"
}

func Hello2() string {
	return quote.Hello()
}
