package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/mvavassori/bare-analytics/models"
)

func GetUsers(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		rows, err := db.Query(`
			SELECT users.id, users.name, users.email, users.password, websites.id, websites.domain, websites.user_id
			FROM users
			LEFT JOIN websites ON users.id = websites.user_id
		`)
		if err != nil {
			log.Println("Error querying users:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		users := make(map[int]*models.User)
		for rows.Next() {
			var user models.User
			var website models.Website
			err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.Password, &website.ID, &website.Domain, &website.UserID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if u, ok := users[user.ID]; ok {
				u.Websites = append(u.Websites, website)
			} else {
				user.Websites = append(user.Websites, website)
				users[user.ID] = &user
			}
		}

		if err := rows.Err(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var usersSlice []models.User
		for _, user := range users {
			usersSlice = append(usersSlice, *user)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(usersSlice)
	}
}
