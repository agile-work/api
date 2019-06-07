package middlewares

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/agile-work/srv-shared/sql-builder/builder"
	"github.com/dgrijalva/jwt-go"

	"github.com/agile-work/srv-mdl-shared/models"
	shared "github.com/agile-work/srv-shared"
	"github.com/agile-work/srv-shared/sql-builder/db"
)

// Authorization validates the token and insert user data in the request
func Authorization(next http.Handler) http.Handler {
	// TODO: Pensar em um jeito talvez de desconsiderar o token e utilizar o certificado p/ garantir a segurança na comunicação dos serviços com a api
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.RequestURI, "/auth/login") {
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

// GetUser set user information to security on request
func GetUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.RequestURI, "/core/instances") {
			userID := r.Header.Get("userID")
			if userID != "" {
				user := models.User{}
				userIDColumn := fmt.Sprintf("%s.id", shared.TableCoreUsers)
				condition := builder.Equal(userIDColumn, userID)
				err := db.LoadStruct(shared.TableCoreUsers, &user, condition)
				if err != nil {
					http.Error(w, http.StatusText(401), http.StatusUnauthorized)
					return
				}
				user.Password = ""

				userByte, err := json.Marshal(user)
				if err != nil {
					http.Error(w, http.StatusText(401), http.StatusUnauthorized)
					return
				}
				r.Header.Add("User", string(userByte))
			}
		}

		next.ServeHTTP(w, r)
	})
}
