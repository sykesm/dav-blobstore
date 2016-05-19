package handlers

import "net/http"

type AuthenticationHandler struct {
	Authorized map[string]string
	PublicRead bool
	Delegate   http.Handler
}

func (ah *AuthenticationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		if ah.PublicRead {
			break
		}
		fallthrough

	default:
		username, password, ok := r.BasicAuth()
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if ah.Authorized == nil || ah.Authorized[username] != password {
			w.WriteHeader(http.StatusForbidden)
			return
		}
	}

	ah.Delegate.ServeHTTP(w, r)
}
