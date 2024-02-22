package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/mvavassori/bare-analytics/utils"
)

// added because of type complains
type contextKey string

const UserIdKey contextKey = "userId"

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("AuthMiddleware called")

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

		// fmt.Println(claims)

		// Add the userId to the context
		ctx := context.WithValue(r.Context(), UserIdKey, userId)

		// This line is responsible for passing the request to the next handler in the chain (e.g., the GetUser function) after the middleware has done its job
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
