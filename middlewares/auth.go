package middlewares

import (
	"net/http"
	"strings"

	"github.com/agile-work/srv-mdl-shared/models"
	"github.com/dgrijalva/jwt-go"
)

// Authorization validates the token and insert user data in the request
func Authorization(next http.Handler) http.Handler {
	// TODO: Pensar em um jeito talvez de desconsiderar o token e utilizar o certificado p/ garantir a segurança na comunicação dos serviços com a api
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.RequestURI, "/auth/login") && r.Method != "OPTIONS" {
			token, err := jwt.ParseWithClaims(r.Header.Get("Authorization"), &models.UserCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
				return []byte("AllYourBase"), nil // TODO: Check the best place for this key, probably the config.toml
			})
			if err != nil {
				http.Error(w, http.StatusText(401), http.StatusUnauthorized)
				return
			}
			if token != nil && token.Valid {
				claims := token.Claims.(*models.UserCustomClaims)
				r.Header.Add("userID", claims.User.ID)
				if r.Header.Get("Content-Language") == "" {
					r.Header.Add("Content-Language", claims.User.LanguageCode)
				}
			} else {
				http.Error(w, http.StatusText(401), http.StatusUnauthorized)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
