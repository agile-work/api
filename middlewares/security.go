package middlewares

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/agile-work/srv-shared/sql-builder/builder"

	"github.com/agile-work/srv-mdl-shared/models"
	shared "github.com/agile-work/srv-shared"
	"github.com/agile-work/srv-shared/sql-builder/db"
)

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
