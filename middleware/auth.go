package middleware

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"strings"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/mvavassori/bare-analytics/utils"
)

// added because of type complains
type contextKey string

const UserIdKey contextKey = "userId"
const RoleKey contextKey = "role"

func AdminOrAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		log.Println("AdminOrAuthMiddleware called")

		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(tokenString, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid Authorization header", http.StatusUnauthorized)
			return
		}

		token, err := utils.ValidateToken(parts[1])
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		if !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		claims := token.Claims.(jwt.MapClaims)
		userId := int(claims["userId"].(float64))
		role := claims["role"].(string)

		// Check if the user is logged in or is an admin
		if userId <= 0 && role != "admin" {
			http.Error(w, "Unauthorized access", http.StatusUnauthorized)
			return
		}

		// Add the userId and role to the context so that the next handler can access them (e.g., the GetUser function)
		ctx := context.WithValue(r.Context(), UserIdKey, userId)
		ctx = context.WithValue(ctx, RoleKey, role)

		// This line is responsible for passing the request to the next handler in the chain (e.g., the GetUser function) after the middleware has done its job
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func AdminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		log.Println("AdminMiddleware called")

		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(tokenString, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid Authorization header", http.StatusUnauthorized)
			return
		}

		token, err := utils.ValidateToken(parts[1])
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		if !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		claims := token.Claims.(jwt.MapClaims)
		role := claims["role"].(string)
		if role != "admin" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func AdminOrOwnerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		log.Println("AdminOrOwnerMiddleware called")

		urlUserID, err := utils.ExtractIDFromURL(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(tokenString, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid Authorization header", http.StatusUnauthorized)
			return
		}

		token, err := utils.ValidateToken(parts[1])
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		if !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		claims := token.Claims.(jwt.MapClaims)
		userId := int(claims["userId"].(float64))
		role := claims["role"].(string)

		// Check if the user is an admin or the owner of the data
		if role != "admin" && userId != urlUserID {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func AdminOrUserWebsiteMiddleware(db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Println("AdminOrUserWebsiteMiddleware called")

			urlWebsiteID, err := utils.ExtractIDFromURL(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			tokenString := r.Header.Get("Authorization")
			if tokenString == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			parts := strings.Split(tokenString, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "Invalid Authorization header", http.StatusUnauthorized)
				return
			}

			token, err := utils.ValidateToken(parts[1])
			if err != nil {
				log.Println(err.Error())
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			if !token.Valid {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			claims := token.Claims.(jwt.MapClaims)
			userID := int(claims["userId"].(float64))
			role := claims["role"].(string)

			// Query the database to get the websites the current user owns
			rows, err := db.Query("SELECT id FROM websites WHERE user_id = $1", userID)
			if err != nil {
				log.Println("Error querying websites:", err)
				http.Error(w, "Error retrieving websites", http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			// Collect the website IDs
			var websiteIDs []int
			for rows.Next() {
				var websiteID int
				err := rows.Scan(&websiteID)
				if err != nil {
					log.Println("Error scanning website:", err)
					http.Error(w, "Error scanning website", http.StatusInternalServerError)
					return
				}
				websiteIDs = append(websiteIDs, websiteID)
			}

			// Check if the user is an admin or the owner of the website
			isAdmin := (role == "admin")
			isOwner := false
			for _, id := range websiteIDs {
				if id == urlWebsiteID {
					isOwner = true
					break
				}
			}

			if !isAdmin && !isOwner {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Proceed to the next handler
			next.ServeHTTP(w, r)
		})
	}
}
