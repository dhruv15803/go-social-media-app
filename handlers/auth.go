package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/dhruv15803/social-media-app/helpers"
	"github.com/dhruv15803/social-media-app/storage"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type RegisterUserRequest struct {
	Email       string `json:"email"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	DateOfBirth string `json:"date_of_birth"`
}

type LoginUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Username string `json:"username"`
}

var (
	MIN_USER_AGE int    = 15
	JWT_SECRET   []byte = []byte(os.Getenv("JWT_SECRET"))
)

func (h *Handler) RegisterUserHandler(w http.ResponseWriter, r *http.Request) {
	var registerUserPayload RegisterUserRequest

	if err := json.NewDecoder(r.Body).Decode(&registerUserPayload); err != nil {
		writeJSONError(w, "invalid request body", http.StatusInternalServerError)
		return
	}

	userEmail := strings.ToLower(strings.TrimSpace(registerUserPayload.Email))
	userUsername := strings.TrimSpace(registerUserPayload.Username)
	userPassword := strings.TrimSpace(registerUserPayload.Password)
	userDateOfBirth := registerUserPayload.DateOfBirth

	userDateOfBirthTime, err := time.Parse("2006-01-02", userDateOfBirth)
	if err != nil {
		writeJSONError(w, "invalid date_of_birth field", http.StatusBadRequest)
		return
	}

	if userEmail == "" || userUsername == "" || userPassword == "" {
		writeJSONError(w, "all fields are required", http.StatusBadRequest)
		return
	}

	if utf8.RuneCountInString(userUsername) < 3 {
		writeJSONError(w, "username should have atleast 3 characters", http.StatusBadRequest)
		return
	}

	if !helpers.IsEmailValid(userEmail) {
		writeJSONError(w, "invalid email format", http.StatusBadRequest)
		return
	}

	if !helpers.IsPasswordStrong(userPassword) {
		writeJSONError(w, "weak password", http.StatusBadRequest)
		return
	}

	if helpers.CalculateAgeFromTime(userDateOfBirthTime) < MIN_USER_AGE {
		writeJSONError(w, "user needs to be atleast age 15 to register", http.StatusBadRequest)
		return
	}

	// registration data validated , now checking if another user with this username or email exists

	existingUsers, err := h.storage.GetUsersByEmailOrUsername(userEmail, userUsername)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Printf("failed to fetch users by email or username :- %v\n", err.Error())
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if len(existingUsers) != 0 {
		writeJSONError(w, "user already exists", http.StatusBadRequest)
		return
	}

	userHashedPassword, err := bcrypt.GenerateFromPassword([]byte(userPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("failed to hash password using bcrypt :- %v\n", err.Error())
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	newUser, err := h.storage.CreateUser(userEmail, userUsername, string(userHashedPassword), userDateOfBirthTime.Format("2006-01-02"))
	if err != nil {
		log.Printf("failed to insert user in db :- %v\n", err.Error())
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	claims := jwt.MapClaims{
		"userId": newUser.Id,
		"exp":    time.Now().Add(time.Hour * 24 * 2).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(JWT_SECRET)
	if err != nil {
		log.Printf("failed to sign token with secret :- %v\n", err.Error())
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	var sameSiteConfig http.SameSite

	if os.Getenv("GO_ENV") == "development" {
		sameSiteConfig = http.SameSiteLaxMode
	} else {
		sameSiteConfig = http.SameSiteNoneMode
	}

	cookie := http.Cookie{
		Name:     "auth_token",
		Value:    signedToken,
		HttpOnly: true,
		Path:     "/",
		Secure:   os.Getenv("GO_ENV") == "production",
		SameSite: sameSiteConfig,
		MaxAge:   60 * 60 * 24 * 2,
	}

	http.SetCookie(w, &cookie)

	type Response struct {
		Success bool         `json:"success"`
		Message string       `json:"message"`
		User    storage.User `json:"user"`
	}

	if err := writeJSON(w, Response{Success: true, Message: "user registered successfully", User: *newUser}, http.StatusCreated); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
	}
}

func (h *Handler) LoginUserHandler(w http.ResponseWriter, r *http.Request) {
	var loginUserPayload LoginUserRequest
	var err error

	if err = json.NewDecoder(r.Body).Decode(&loginUserPayload); err != nil {
		writeJSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if loginUserPayload.Email == "" && loginUserPayload.Username == "" {
		writeJSONError(w, "either email or username is required , both cannot be empty", http.StatusBadRequest)
		return
	}

	var user *storage.User
	isLoginByEmail := false

	if loginUserPayload.Email != "" {

		isLoginByEmail = true

		userEmail := strings.ToLower(strings.TrimSpace(loginUserPayload.Email))
		// login by email
		user, err = h.storage.GetUserByEmail(userEmail)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeJSONError(w, "invalid email or password", http.StatusBadRequest)
				return
			} else {
				log.Printf("failed to get user by email from db :- %v\n", err.Error())
				writeJSONError(w, "internal server error", http.StatusInternalServerError)
				return
			}
		}
	} else {
		// login by username
		userUsername := strings.TrimSpace(loginUserPayload.Username)

		user, err = h.storage.GetUserByUsername(userUsername)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeJSONError(w, "invalid username or password", http.StatusBadRequest)
				return
			} else {
				log.Printf("failed to get user by username from db :- %v\n", err.Error())
				writeJSONError(w, "internal server error", http.StatusInternalServerError)
				return
			}
		}
	}

	userPlainPassword := strings.TrimSpace(loginUserPayload.Password)

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(userPlainPassword)); err != nil {
		var errMessage string

		if isLoginByEmail {
			errMessage = "invalid email or password"
		} else {
			errMessage = "invalid username or password"
		}

		writeJSONError(w, errMessage, http.StatusBadRequest)
		return
	}

	// valid  credentials
	claims := jwt.MapClaims{
		"userId": user.Id,
		"exp":    time.Now().Add(time.Hour * 24 * 2).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString(JWT_SECRET)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	var sameSiteConfig http.SameSite

	if os.Getenv("GO_ENV") == "development" {
		sameSiteConfig = http.SameSiteLaxMode
	} else {
		sameSiteConfig = http.SameSiteNoneMode
	}

	cookie := http.Cookie{
		Name:     "auth_token",
		Value:    signedToken,
		HttpOnly: true,
		Path:     "/",
		Secure:   os.Getenv("GO_ENV") == "production",
		SameSite: sameSiteConfig,
		MaxAge:   60 * 60 * 24 * 2,
	}

	http.SetCookie(w, &cookie)

	type Response struct {
		Success bool         `json:"success"`
		Message string       `json:"message"`
		User    storage.User `json:"user"`
	}

	if err := writeJSON(w, Response{Success: true, Message: "user logged in successfully", User: *user}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
	}
}

func (h *Handler) GetAuthUserHandler(w http.ResponseWriter, r *http.Request) {
	// auth middleware activated
	// get userId from request context

	userId, ok := r.Context().Value(AuthUserId).(int)
	if !ok {
		log.Println("AuthUserId not of type int")
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	user, err := h.storage.GetUserById(userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "user not found", http.StatusBadRequest)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			// return
		}
	}

	type Response struct {
		Success bool         `json:"success"`
		User    storage.User `json:"user"`
	}

	if err := writeJSON(w, Response{Success: true, User: *user}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
}
