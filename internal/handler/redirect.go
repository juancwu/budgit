package handler

import "net/http"

type redirectHandler struct{}

func NewRedirectHandler() *redirectHandler {
	return &redirectHandler{}
}

func (h *redirectHandler) Spaces(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/app/spaces", http.StatusMovedPermanently)
}
