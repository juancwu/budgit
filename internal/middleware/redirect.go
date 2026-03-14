package middleware

import "net/http"

func Redirect(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		redirect(w, r, path, http.StatusSeeOther)
	}
}
