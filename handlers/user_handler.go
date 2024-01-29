package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
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
				// If the website ID is not null, append the website to the user's Websites slice
				if website.ID.Valid {
					u.Websites = append(u.Websites, website)
				}
			} else {
				// If the website ID is not null, append the website to the user's Websites slice
				if website.ID.Valid {
					user.Websites = append(user.Websites, website)
				}
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

func GetUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, ok := vars["id"]

		if !ok {
			http.Error(w, "ID not provided in the URL", http.StatusBadRequest)
			return
		}

		// Validate that the ID is a number
		_, err := strconv.Atoi(id)
		if err != nil {
			http.Error(w, "ID must be a number", http.StatusBadRequest)
			return
		}

		rows, err := db.Query(`
            SELECT users.id, users.name, users.email, users.password, websites.id, websites.domain, websites.user_id
            FROM users
            LEFT JOIN websites ON users.id = websites.user_id
            WHERE users.id = $1
        `, id)

		if err != nil {
			log.Println("Error retrieving user and websites:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var user models.User
		user.Websites = make([]models.Website, 0)

		found := false // Flag to check if any rows were returned

		for rows.Next() {
			found = true
			var website models.Website
			err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.Password, &website.ID, &website.Domain, &website.UserID)
			if err != nil {
				log.Println("Error scanning user and website:", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			// If the website ID is not null, append the website to the user's Websites slice
			if website.ID.Valid {
				user.Websites = append(user.Websites, website)
			}
		}

		if !found {
			http.Error(w, fmt.Sprintf("User with ID %s not found", id), http.StatusNotFound)
			return
		}

		jsonResponse, err := json.Marshal(user)
		if err != nil {
			log.Println("Error encoding JSON:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResponse)
	}
}

func CreateUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user models.UserInsert

		// Decode the JSON in the request body into the user struct
		err := json.NewDecoder(r.Body).Decode(&user)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		// Validate the user struct
		err = user.Validate()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Insert the user into the database
		_, err = db.Exec(`
			INSERT INTO users (name, email, password)
			VALUES ($1, $2, $3)
		`, user.Name, user.Email, user.Password)

		if err != nil {
			log.Println("Error inserting user:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}

func UpdateUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, ok := vars["id"]

		if !ok {
			http.Error(w, "ID not provided in the URL", http.StatusBadRequest)
			return
		}

		// Validate that the ID is a number
		_, err := strconv.Atoi(id)
		if err != nil {
			http.Error(w, "ID must be a number", http.StatusBadRequest)
			return
		}

		var user models.User
		err = json.NewDecoder(r.Body).Decode(&user)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		_, err = db.Exec(`
            UPDATE users
            SET name = $1, email = $2, password = $3
            WHERE id = $4
        `, user.Name, user.Email, user.Password, id)

		if err != nil {
			log.Println("Error updating user:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func DeleteUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, ok := vars["id"]

		if !ok {
			http.Error(w, "ID not provided in the URL", http.StatusBadRequest)
			return
		}

		// Validate that the ID is a number
		_, err := strconv.Atoi(id)
		if err != nil {
			http.Error(w, "ID must be a number", http.StatusBadRequest)
			return
		}

		_, err = db.Exec(`
            DELETE FROM users
            WHERE id = $1
        `, id)

		if err != nil {
			log.Println("Error deleting user:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
