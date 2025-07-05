package handlers

import (
	"context"
	"log"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
)

var (
	AuthUserId = "userId"
)

func (h *Handler) AuthMiddleware(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		cookie, err := r.Cookie("auth_token")
		if err != nil {
			log.Printf("failed to extract cookie from request :- %v\n", err.Error())
			writeJSONError(w, "user auth_token not available", http.StatusBadRequest)
			return
		}

		tokenStr := cookie.Value

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			return JWT_SECRET, nil
		})

		if err != nil {
			log.Printf("failed to parse token :- %v\n", err.Error())
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			log.Println("claims not of type jwt MapClaims")
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if !token.Valid {
			log.Println("invalid token")
			writeJSONError(w, "invalid token", http.StatusBadRequest)
			return
		}

		userIdFloat, ok := claims["userId"].(float64)
		if !ok {
			log.Println("userId in claims is not of type float64")
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		userId := int(userIdFloat)

		ctx := context.WithValue(r.Context(), AuthUserId, userId)
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}
