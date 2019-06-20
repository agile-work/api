package middlewares

import (
	"net/http"
	"strings"

	"github.com/agile-work/srv-shared/token"
)

// Authorization validates the token and insert user data in the request
func Authorization(next http.Handler) http.Handler {
	// TODO: Pensar em um jeito talvez de desconsiderar o token e utilizar o certificado p/ garantir a segurança na comunicação dos serviços com a api
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.RequestURI, "/auth/login") {
			payload, err := token.Validate(r.Header.Get("Authorization"))
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			if value, ok := payload["code"]; ok {
				r.Header.Add("username", value.(string))
			} else {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			if value, ok := payload["user_id"]; ok {
				r.Header.Add("userID", value.(string))
			} else {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			if value, ok := payload["language_code"]; ok && r.Header.Get("Content-Language") == "" {
				r.Header.Add("Content-Language", value.(string))
			}
		}

		next.ServeHTTP(w, r)
	})
}
