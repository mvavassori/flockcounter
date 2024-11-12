package middleware

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"strings"

	"github.com/mvavassori/bare-analytics/utils"
)

// added because of type complains
type contextKey string

const UserIdKey contextKey = "userId"
const RoleKey contextKey = "role"

func AdminOrAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		// Validate the token and extract claims
		claims, err := utils.ValidateTokenAndExtractClaims(parts[1])
		if err != nil {
			log.Println("Token validation error:", err)
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Extract user ID and role from claims
		userId, ok := claims["userId"].(float64)
		if !ok {
			http.Error(w, "Invalid token payload", http.StatusUnauthorized)
			return
		}
		role, ok := claims["role"].(string)
		if !ok {
			http.Error(w, "Invalid token payload", http.StatusUnauthorized)
			return
		}

		// Check if the user is logged in or is an admin
		if role != "admin" && int(userId) <= 0 {
			http.Error(w, "Unauthorized access", http.StatusUnauthorized)
			return
		}

		// Add userId and role to context
		ctx := context.WithValue(r.Context(), UserIdKey, int(userId))
		ctx = context.WithValue(ctx, RoleKey, role)

		// This line is responsible for passing the request to the next handler in the chain (e.g., the GetUser function) after the middleware has done its job.
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func Admin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		// Validate the token and extract claims
		claims, err := utils.ValidateTokenAndExtractClaims(parts[1])
		if err != nil {
			log.Println("Token validation error:", err)
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		role := claims["role"].(string)
		if role != "admin" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func AdminOrOwner(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		// Validate the token and extract claims
		claims, err := utils.ValidateTokenAndExtractClaims(parts[1])
		if err != nil {
			log.Println("Token validation error:", err)
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

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

// check for website domain
func AdminOrUserWebsite(db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			urlWebsiteDomain, err := utils.ExtractDomainFromURL(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			// Check if the domain exists in the database
			var domainExists bool
			err = db.QueryRow("SELECT EXISTS (SELECT 1 FROM websites WHERE domain = $1)", urlWebsiteDomain).Scan(&domainExists)
			if err != nil {
				log.Println("Error checking domain existence:", err)
				http.Error(w, "Error checking domain", http.StatusInternalServerError)
				return
			}

			// If domain doesn't exist, return a 404 Not Found
			if !domainExists {
				http.Error(w, "Website not found", http.StatusNotFound)
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

			// Validate the token and extract claims
			claims, err := utils.ValidateTokenAndExtractClaims(parts[1])
			if err != nil {
				log.Println("Token validation error:", err)
				http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
				return
			}

			userID := int(claims["userId"].(float64))
			role := claims["role"].(string)

			// Query the database to get the websites the current user owns
			rows, err := db.Query("SELECT domain FROM websites WHERE user_id = $1", userID)
			if err != nil {
				log.Println("Error querying websites:", err)
				http.Error(w, "Error retrieving websites", http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			var websiteDomains []string
			for rows.Next() {
				var websiteDomain string
				err := rows.Scan(&websiteDomain)
				if err != nil {
					log.Println("Error scanning website:", err)
					http.Error(w, "Error scanning website", http.StatusInternalServerError)
					return
				}
				websiteDomains = append(websiteDomains, websiteDomain)
			}

			// Check if the user is an admin or the owner of the website
			isAdmin := (role == "admin")
			isOwner := false
			for _, domain := range websiteDomains {
				if domain == urlWebsiteDomain {
					isOwner = true
					break
				}
			}

			if len(websiteDomains) == 0 && !isAdmin {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
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
