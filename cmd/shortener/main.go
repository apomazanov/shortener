package main

import (
	"net/http"

	"github.com/apomazanov/shortener/internal/handler"
	"github.com/apomazanov/shortener/internal/repository"
)

/* -------------------------------------------------------------------------- */
func main() {
	repo := repository.NewInMemoryRepo();
	handler := handler.New(repo)
	
	mux := http.NewServeMux()
	mux.HandleFunc(`GET /{id}`, handler.ExtractURL)
	mux.HandleFunc(`POST /`, handler.RegisterURL)
	mux.HandleFunc(`/{path...}`, handler.Reject)

	err := http.ListenAndServe(`127.0.0.1:8080`, mux)

	if err != nil {
		panic(err)
	}
}
