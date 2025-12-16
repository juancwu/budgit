package handler

import "net/http"

type authHandler struct {
}

func NewAuthHandler() *authHandler {
	return &authHandler{}
}

func (h *authHandler) AuthPage(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("200 OK"))
}
