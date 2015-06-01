package main

import "net/http"

func UsedHandlerFunc(w http.ResponseWriter, r *http.Request) {

}

func UnusedHandlerFunc(w http.ResponseWriter, r *http.Request) {

}

type Foo int

func (f Foo) UsedInAnon() int { return 0 }

func main() {

	foo := Foo(9)
	f := func() {
		foo.UsedInAnon()
	}
	f()

	http.HandleFunc("/", UsedHandlerFunc)
}
