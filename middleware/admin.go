package middleware

import (
	// "log"
	"database/sql"
	"log"
	"net/http"
)

func AdminMiddleware(db *sql.DB) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Println("AdminMiddleware called")
			// Get the userId from the context
			userId := r.Context().Value(UserIdKey)
			if userId == nil {
				http.Error(w, "Authorization required", http.StatusUnauthorized)
				return
			}

			// Query the database to check if the user is an admin
			var role string
			err := db.QueryRow("SELECT role FROM users WHERE id = $1", userId).Scan(&role)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			// If the user is not an admin, return an error
			if role != "admin" {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			// If the user is an admin, call the next handler
			next.ServeHTTP(w, r)
		})
	}
}
