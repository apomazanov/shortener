package handler

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/apomazanov/shortener/internal/repository"
)

func Get(w http.ResponseWriter, req *http.Request) {
	short := strings.TrimPrefix(req.URL.Path, "/")
	long, ok := repository.GetRecord(short)
	if !ok {
		http.Error(w, "Error finding URL", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Location", long)
	w.WriteHeader(http.StatusTemporaryRedirect)
	fmt.Fprintf(w, "Redirecting to %s\n", long)
}

func Post(w http.ResponseWriter, req *http.Request) {
	/*
		if !strings.HasPrefix(req.Header.Get("Content-Type"), "text/plain") {
			http.Error(w, "Only text/plain allowed", http.StatusBadRequest)
			return
		}
	*/

	defer req.Body.Close()
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	short, ok := repository.AddRecord(string(body))
	if !ok {
		http.Error(w, "Error appending storage", http.StatusBadRequest)
		return
	}
	respBody := fmt.Sprintf("http://localhost:8080/%s", short)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(respBody))
}

func CatchAll(w http.ResponseWriter, req *http.Request) {
	fmt.Println("Not supported")
	http.Error(w, "", http.StatusBadRequest)
}
