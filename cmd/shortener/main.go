package main

import (
	"net/http"

	"github.com/apomazanov/shortener/internal/handler"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc(`GET /{id}`, handler.Get)
	mux.HandleFunc(`POST /`, handler.Post)
	mux.HandleFunc(`/{path...}`, handler.CatchAll)

	err := http.ListenAndServe(`127.0.0.1:8080`, mux)

	if err != nil {
		panic(err)
	}
}
