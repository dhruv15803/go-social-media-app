package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
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
	"github.com/google/uuid"
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

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

type ResetUserPasswordRequest struct {
	Password string `json:"password"`
}

var (
	MIN_USER_AGE                  int    = 15
	JWT_SECRET                    []byte = []byte(os.Getenv("JWT_SECRET"))
	VERIFICATION_MAIL_RETRY_COUNT        = 3
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

	// check if active users with username or email exists in the db
	activeUsers, err := h.storage.GetActiveUsersByEmailOrUsername(userEmail, userUsername)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Printf("failed to get active users by email or username :- %v\n", err.Error())
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if len(activeUsers) != 0 {
		writeJSONError(w, "user already exists", http.StatusBadRequest)
		return
	}

	// if not , continue with the process
	// hash plain text password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(userPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("failed to hash password :- %v\n", err.Error())
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// create a unique verification token for the user
	verificationTokenStr := uuid.New().String()
	expirationTime := time.Now().Add(time.Hour * 24 * 3)

	// create user along with its corresponding invitation  using a transaction
	newUser, err := h.storage.CreateUserAndInvitation(userEmail, userUsername, string(hashedPassword), userDateOfBirth, verificationTokenStr, expirationTime)
	if err != nil {
		log.Printf("failed to create user and invitation :- %v\n", err.Error())
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// send mail

	if err := helpers.SendVerificationMailWithRetry(os.Getenv("GOMAIL_FROM_EMAIL"), "Activate account", *newUser, verificationTokenStr, "./templates/verification.html", VERIFICATION_MAIL_RETRY_COUNT); err != nil {
		log.Printf("failed to send mail :- %v\n", err.Error())
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	type Response struct {
		Success bool         `json:"success"`
		Message string       `json:"message"`
		User    storage.User `json:"user"`
	}

	if err := writeJSON(w, Response{Success: true, Message: "registered user successfully and sent verification mail", User: *newUser}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) ActivateUserHandler(w http.ResponseWriter, r *http.Request) {

	tokenStr := r.URL.Query().Get("token")

	activeUser, err := h.storage.ActivateUser(tokenStr)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	claims := jwt.MapClaims{
		"userId": activeUser.Id,
		"exp":    time.Now().Add(time.Hour * 24 * 30).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	jwtTokenStr, err := token.SignedString(JWT_SECRET)
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
		Value:    jwtTokenStr,
		HttpOnly: true,
		Path:     "/",
		Secure:   os.Getenv("GO_ENV") == "production",
		SameSite: sameSiteConfig,
		MaxAge:   60 * 60 * 24 * 30,
	}

	http.SetCookie(w, &cookie)

	type Response struct {
		Success bool         `json:"success"`
		Message string       `json:"message"`
		User    storage.User `json:"user"`
	}

	if err := writeJSON(w, Response{Success: true, Message: "user activated", User: *activeUser}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
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
		user, err = h.storage.GetActiveUserByEmail(userEmail)
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

		user, err = h.storage.GetActiveUserByUsername(userUsername)
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
		"exp":    time.Now().Add(time.Hour * 24 * 30).Unix(),
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
		MaxAge:   60 * 60 * 24 * 30,
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

func (h *Handler) LogoutUserHandler(w http.ResponseWriter, r *http.Request) {

	userId, ok := r.Context().Value(AuthUserId).(int)
	if !ok {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	_, err := h.storage.GetUserById(userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "user not found", http.StatusBadRequest)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	var sameSiteConfig http.SameSite

	if os.Getenv("GO_ENV") == "development" {
		sameSiteConfig = http.SameSiteLaxMode
	} else {
		sameSiteConfig = http.SameSiteNoneMode
	}
	cookie := http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Secure:   os.Getenv("GO_ENV") == "production",
		HttpOnly: true,
		SameSite: sameSiteConfig,
	}

	http.SetCookie(w, &cookie)

	type Response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	if err := writeJSON(w, Response{Success: true, Message: "logged out successfully"}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
	}
}

func (h *Handler) ForgotPasswordHandler(w http.ResponseWriter, r *http.Request) {

	var forgotPasswordPayload ForgotPasswordRequest

	if err := json.NewDecoder(r.Body).Decode(&forgotPasswordPayload); err != nil {
		writeJSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	userEmail := strings.ToLower(strings.TrimSpace(forgotPasswordPayload.Email))

	if userEmail == "" {
		writeJSONError(w, "email is required", http.StatusBadRequest)
		return
	}

	if !helpers.IsEmailValid(userEmail) {
		writeJSONError(w, "invalid email", http.StatusBadRequest)
		return
	}

	activeUser, err := h.storage.GetActiveUserByEmail(userEmail)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "user account not found", http.StatusBadRequest)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	plainToken, tokenHash, err := helpers.GenerateToken()
	if err != nil {
		log.Printf("failed to generate token :- %v\n", err.Error())
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	passwordResetExpirationTime := time.Now().Add(time.Minute * 15)

	if err := h.storage.CreatePasswordResetForUser(tokenHash, activeUser.Id, passwordResetExpirationTime); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	maxRetryCount := 3
	if err := helpers.SendPasswordResetMailWithRetry(os.Getenv("GOMAIL_FROM_EMAIL"), "Password reset", *activeUser, plainToken, "./templates/passwordReset.html", maxRetryCount); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	type Response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	if err := writeJSON(w, Response{Success: true, Message: ""}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) ResetUserPasswordHandler(w http.ResponseWriter, r *http.Request) {

	var resetUserPasswordPayload ResetUserPasswordRequest

	if err := json.NewDecoder(r.Body).Decode(&resetUserPasswordPayload); err != nil {
		writeJSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	newPassword := strings.TrimSpace(resetUserPasswordPayload.Password)

	plainToken := r.URL.Query().Get("token")

	hashTokenBytes := sha256.Sum256([]byte(plainToken))

	hashedToken := hex.EncodeToString(hashTokenBytes[:])

	if newPassword == "" {
		writeJSONError(w, "password is required", http.StatusBadRequest)
		return
	}

	if !helpers.IsPasswordStrong(newPassword) {
		writeJSONError(w, "weak password", http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if err := h.storage.ResetPassword(string(hashedPassword), hashedToken); err != nil {
		log.Printf("failed to reset password :- %v\n", err.Error())
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	type Response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	if err := writeJSON(w, Response{Success: true, Message: "reset password successfully"}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
}
