package middleware

import "net/http"

func Redirect(path string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("HX-Request") == "true" {
			w.Header().Set("HX-Redirect", path)
			w.WriteHeader(http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, path, http.StatusSeeOther)
	})
}
