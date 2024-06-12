package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	// "github.com/mvavassori/bare-analytics/middleware"
	"github.com/mvavassori/bare-analytics/models"
	"github.com/mvavassori/bare-analytics/utils"
	"golang.org/x/crypto/bcrypt"
)

// CRUD operations for users

func GetUsers(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		rows, err := db.Query(`
			SELECT users.id, users.name, users.email, users.password, users.role, users.created_at, users.updated_at , websites.id, websites.domain, websites.user_id
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
			err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.Password, &user.Role, &user.CreatedAt, &user.UpdatedAt, &website.ID, &website.Domain, &website.UserID)
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

		// // Extract the userId from the context
		// tokenUserID := r.Context().Value(middleware.UserIdKey).(int)

		// // Compare the userId in the context with the userId in the request
		// if id != tokenUserID {
		// 	http.Error(w, "Unauthorized", http.StatusUnauthorized)
		// 	return
		// }

		rows, err := db.Query(`
            SELECT users.id, users.name, users.email, users.password, users.role, users.created_at, users.updated_at, websites.id, websites.domain, websites.user_id
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
			err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.Password, &user.Role, &user.CreatedAt, &user.UpdatedAt, &website.ID, &website.Domain, &website.UserID)
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

func CreateUser(db *sql.DB, isAdmin bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user models.UserInsert

		// Decode the JSON in the request body into the user struct
		err := json.NewDecoder(r.Body).Decode(&user)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		// Set the role based on the isAdmin parameter
		if isAdmin {
			user.Role = "admin"
		} else {
			user.Role = "user"
		}

		// Validate the user struct
		err = user.Validate()
		if err != nil {
			// http.Error(w, err.Error(), http.StatusBadRequest)
			utils.WriteErrorResponse(w, http.StatusBadRequest, errors.New(err.Error()))
			return
		}

		// Check if a user with the same email already exists in the database
		var existingEmail string
		err = db.QueryRow(`
			SELECT email
			FROM users
			WHERE email = $1
		`, user.Email).Scan(&existingEmail)

		if err == nil {
			// If a user with the same email already exists, return a conflict error
			// http.Error(w, "Conflict", http.StatusConflict)
			utils.WriteErrorResponse(w, http.StatusConflict, errors.New("Conflict"))
			return
		} else if err != sql.ErrNoRows {
			// If there was an error executing the query, return an internal server error
			log.Println("Error checking for existing email:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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

		now := time.Now()

		var userID int
		// Insert the user into the database and return the ID of the newly inserted user
		err = db.QueryRow(`
            INSERT INTO users (name, email, password, role, created_at, updated_at)
            VALUES ($1, $2, $3, $4, $5, $6)
            RETURNING id
        `, user.Name, user.Email, string(hashedPassword), user.Role, now, now).Scan(&userID)

		if err != nil {
			log.Println("Error inserting user:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Generate a token for the new user
		// tokenString, err := utils.CreateToken(int(userID))
		// if err != nil {
		// 	log.Println("Error creating token:", err)
		// 	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		// 	return
		// }

		// fmt.Println("Token:", tokenString)

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

		var userUpdate models.UserUpdate
		err = json.NewDecoder(r.Body).Decode(&userUpdate)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		now := time.Now()

		result, err := db.Exec(`
			UPDATE users
			SET name = $1, email = $2, password = $3, updated_at = $4
			WHERE id = $5
		`, userUpdate.Name, userUpdate.Email, userUpdate.Password, now, id)

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

// User authentication

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
			// http.Error(w, err.Error(), http.StatusBadRequest)
			utils.WriteErrorResponse(w, http.StatusBadRequest, errors.New(err.Error()))
			return
		}

		// Get the user's ID, role, and hashed password from the database
		var id int
		var hashedPassword string
		var role string
		var name string
		var email string
		err = db.QueryRow("SELECT id, password, role, name, email FROM users WHERE email = $1", user.Email).Scan(&id, &hashedPassword, &role, &name, &email)
		if err != nil {
			log.Println("Error getting user:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Compare the hashed password with the plain text password
		err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(user.Password))
		if err != nil {
			// http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			utils.WriteErrorResponse(w, http.StatusUnauthorized, errors.New("invalid credentials"))
			return
		}

		// If the passwords match, generate an access token and a refresh token for the user
		accessToken, err := utils.CreateAccessToken(id, role, name, email)
		if err != nil {
			log.Println("Error creating access token:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		refreshToken, err := utils.CreateRefreshToken(id)
		if err != nil {
			log.Println("Error creating refresh token:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Invalidate any existing refresh tokens for the user
		_, err = db.Exec("DELETE FROM refresh_tokens WHERE user_id = $1", id)
		if err != nil {
			log.Println("Error invalidating refresh tokens:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Store the refresh token in the database
		_, err = db.Exec("INSERT INTO refresh_tokens (token, user_id, expires_at) VALUES ($1, $2, $3)", refreshToken, id, time.Now().Add(time.Hour*24*7))
		if err != nil {
			log.Println("Error storing refresh token:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		backendTokens := map[string]interface{}{
			"accessToken":  accessToken,
			"refreshToken": refreshToken,
			"expiresAt":    time.Now().Add(time.Second * 15).Unix(),
		}

		userData := map[string]interface{}{
			"name":  name,
			"email": email,
			"id":    id,
		}

		data := map[string]interface{}{
			"backendTokens": backendTokens,
			"user":          userData,
		}

		response, err := json.Marshal(data)
		if err != nil {
			log.Println("Error encoding JSON:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Set response headers and write the JSON response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(response)
	}
}

func RefreshToken(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the refresh token from the request

		tokenString := r.Header.Get("Authorization")

		if tokenString == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(tokenString, " ")
		if len(parts) != 2 || parts[0] != "Refresh" {
			http.Error(w, "Invalid Authorization header", http.StatusUnauthorized)
			return
		}

		refreshToken := parts[1]

		// Look up the refresh token in the database
		var userID int
		var expirationTime time.Time
		err := db.QueryRow("SELECT user_id, expires_at FROM refresh_tokens WHERE token = $1", refreshToken).Scan(&userID, &expirationTime)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusUnauthorized, errors.New("invalid refresh token"))
			return
		}

		// Check if the refresh token is expired
		if time.Now().After(expirationTime) {
			utils.WriteErrorResponse(w, http.StatusUnauthorized, errors.New("refresh token expired"))
			return
		}

		// Fetch the role of the user from the database
		var role string
		var name string
		var email string
		err = db.QueryRow("SELECT role, name, email FROM users WHERE id = $1", userID).Scan(&role, &name, &email)
		if err != nil {
			http.Error(w, "Error fetching user role", http.StatusInternalServerError)
			return
		}

		// Generate a new access token
		accessToken, err := utils.CreateAccessToken(userID, role, name, email)
		if err != nil {
			http.Error(w, "Error creating access token", http.StatusInternalServerError)
			return
		}

		// accessToken = fmt.Sprintf(`{"accessToken": "%s"}`, accessToken)

		// interface{} means we can use any type
		data := map[string]interface{}{
			"accessToken":  accessToken,
			"refreshToken": refreshToken,
			"expiresAt":    time.Now().Add(time.Minute * 15).Unix(),
		}

		response, err := json.Marshal(data)
		if err != nil {
			log.Println("Error encoding JSON:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Send the new access token to the client
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(response)
	}
}
