package handler

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

type URLRepository interface {
	Add(long string) (string, bool)
	Get(short string) (string, bool)
}

type Handler struct {
	repo URLRepository
}

/* -------------------------------------------------------------------------- */
func New(repo URLRepository) *Handler {
	return &Handler{repo: repo}
}

/* -------------------------------------------------------------------------- */
func (h *Handler) ExtractURL(w http.ResponseWriter, req *http.Request) {
	short := strings.TrimPrefix(req.URL.Path, "/")

	long, ok := h.repo.Get(short)
	if !ok {
		http.Error(w, "URL not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Location", long)
	w.WriteHeader(http.StatusTemporaryRedirect)
	fmt.Fprintf(w, "Redirecting to %s\n", long)
}

/* -------------------------------------------------------------------------- */
func (h *Handler) RegisterURL(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	body, err := io.ReadAll(req.Body)
	if err != nil {
		fmt.Println(err.Error())
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	short, ok := h.repo.Add(string(body))
	if !ok {
		http.Error(w, "Adding failed", http.StatusServiceUnavailable)
		return
	}
	respBody := fmt.Sprintf("http://localhost:8080/%s", short)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(respBody))
}

/* -------------------------------------------------------------------------- */
func (h *Handler) Reject(w http.ResponseWriter, req *http.Request) {
	http.Error(w, "", http.StatusBadRequest)
}
