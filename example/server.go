package main

import (
	"log"
	"net/http"
)

type server int

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL.Path)
	_, _ = w.Write([]byte("Hello zcache\n"))
}

func main() {
	var s server
	_ = http.ListenAndServe(":9999", &s)
}
