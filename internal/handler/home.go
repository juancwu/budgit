package handler

import "net/http"

type homeHandler struct{}

func NewHomeHandler() *homeHandler {
	return &homeHandler{}
}

func (home *homeHandler) NotFoundPage(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("404 Page Not Found"))
}
