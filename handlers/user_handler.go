package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/mvavassori/bare-analytics/models"
	"github.com/mvavassori/bare-analytics/utils"
	"golang.org/x/crypto/bcrypt"
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

		// w.Header().Set("Content-Type", "application/json")
		// json.NewEncoder(w).Encode(usersSlice)

		// Now, 'userSlice' contains all the retrieved users
		jsonResponse, err := json.Marshal(usersSlice)
		if err != nil {
			log.Println("Error encoding JSON:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Set response headers and write the JSON response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResponse)
	}
}

func GetUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := utils.ExtractIDFromURL(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
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
			http.Error(w, fmt.Sprintf("User with id %d doesn't exist", id), http.StatusNotFound)
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

		// Hash the password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Println("Error hashing password:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Insert the user into the database
		// result, err := db.Exec(`
		// 	INSERT INTO users (name, email, password)
		// 	VALUES ($1, $2, $3)
		// `, user.Name, user.Email, user.Password)

		var userID int
		// Insert the user into the database and return the ID of the newly inserted user
		err = db.QueryRow(`
            INSERT INTO users (name, email, password)
            VALUES ($1, $2, $3)
            RETURNING id
        `, user.Name, user.Email, string(hashedPassword)).Scan(&userID)

		if err != nil {
			log.Println("Error inserting user:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Generate a token for the new user
		tokenString, err := utils.CreateToken(int(userID))
		if err != nil {
			log.Println("Error creating token:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		fmt.Println("Token:", tokenString)

		w.WriteHeader(http.StatusCreated)
	}
}

func UpdateUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := utils.ExtractIDFromURL(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var user models.User
		err = json.NewDecoder(r.Body).Decode(&user)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		result, err := db.Exec(`
            UPDATE users
            SET name = $1, email = $2, password = $3
            WHERE id = $4
        `, user.Name, user.Email, user.Password, id)

		if err != nil {
			log.Println("Error updating user:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Check if the user was found
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if rowsAffected == 0 {
			http.Error(w, fmt.Sprintf("User with id %d doesn't exist", id), http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func DeleteUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := utils.ExtractIDFromURL(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		result, err := db.Exec(`
            DELETE FROM users
            WHERE id = $1
        `, id)

		if err != nil {
			log.Println("Error deleting user:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Check if the user was found
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if rowsAffected == 0 {
			http.Error(w, fmt.Sprintf("User with id %d doesn't exist", id), http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func Login(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user models.UserLogin

		// Decode the JSON in the request body into the user struct
		err := json.NewDecoder(r.Body).Decode(&user)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		// Validate the user struct
		err = user.ValidateLogin()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Get the user's ID and hashed password from the database
		var id int
		var hashedPassword string
		err = db.QueryRow("SELECT id, password FROM users WHERE email = $1", user.Email).Scan(&id, &hashedPassword)
		if err != nil {
			log.Println("Error getting user:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Compare the hashed password with the plain text password
		err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(user.Password))
		if err != nil {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		// If the passwords match, generate a token for the user
		tokenString, err := utils.CreateToken(id)
		if err != nil {
			log.Println("Error creating token:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		fmt.Println("Token:", tokenString)

		token, err := json.Marshal(map[string]string{"token": tokenString})
		if err != nil {
			log.Println("Error encoding JSON:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Set response headers and write the JSON response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(token)
	}
}
