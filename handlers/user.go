package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/dhruv15803/social-media-app/storage"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) FollowRequestHandler(w http.ResponseWriter, r *http.Request) {
	userId, ok := r.Context().Value(AuthUserId).(int)
	if !ok {
		log.Println("user id not of type int when asserting as int")
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	user, err := h.storage.GetUserById(userId)
	if err != nil {
		writeJSONError(w, "user not found", http.StatusBadRequest)
		return
	}

	requestReceiverId, err := strconv.Atoi(chi.URLParam(r, "userId"))
	if err != nil {
		writeJSONError(w, "invalid request parameter", http.StatusBadRequest)
		return
	}

	requestReceiver, err := h.storage.GetUserById(requestReceiverId)
	if err != nil {
		writeJSONError(w, "request receiver user not found", http.StatusBadRequest)
		return
	}

	if requestReceiver.Id == user.Id {
		writeJSONError(w, "user cannot send follow request to itself", http.StatusBadRequest)
		return
	}

	if requestReceiver.IsPublic {
		writeJSONError(w, "request receiver is a public account , follow request not required", http.StatusBadRequest)
		return
	}

	follow, _ := h.storage.GetFollow(user.Id, requestReceiver.Id)
	if follow != nil {
		writeJSONError(w, "already following this user", http.StatusBadRequest)
		return
	}

	// if here then private account
	// check if user has already sent a follow request
	existingFollowRequest, err := h.storage.GetFollowRequest(user.Id, requestReceiver.Id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Printf("failed to get follow request")
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if existingFollowRequest == nil {
		followRequest, err := h.storage.CreateFollowRequest(user.Id, requestReceiver.Id)
		if err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		type Response struct {
			Success       bool                  `json:"success"`
			Message       string                `json:"message"`
			FollowRequest storage.FollowRequest `json:"followRequest"`
		}

		if err := writeJSON(w, Response{Success: true, Message: "sent follow request", FollowRequest: *followRequest}, http.StatusCreated); err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	} else {

		if err := h.storage.RemoveFollowRequest(existingFollowRequest.RequestSenderId, existingFollowRequest.RequestReceiverId); err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		type Response struct {
			Success bool   `json:"success"`
			Message string `json:"message"`
		}

		if err := writeJSON(w, Response{Success: true, Message: "removed follow request"}, http.StatusCreated); err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}
}

func (h *Handler) FollowUserHandler(w http.ResponseWriter, r *http.Request) {
	userId, ok := r.Context().Value(AuthUserId).(int)
	if !ok {
		log.Println("user id not of type int when asserting as int")
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	user, err := h.storage.GetUserById(userId)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "user not found", http.StatusBadRequest)
			return
		} else {
			writeJSONError(w, "internal server", http.StatusInternalServerError)
			return
		}
	}

	userToBeFollowedId, err := strconv.Atoi(chi.URLParam(r, "userId"))

	if err != nil {
		writeJSONError(w, "invalid request parameter", http.StatusBadRequest)
		return
	}

	userToBeFollowed, err := h.storage.GetUserById(userToBeFollowedId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "user to be followed not found", http.StatusBadRequest)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	// if user is already following the userToBeFollowed -> remove follow
	// if not following -> create follow (a follow can only be created directly without a request
	// when the userToBeFollowed is a public account)

	existingFollow, err := h.storage.GetFollow(user.Id, userToBeFollowed.Id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if existingFollow == nil {

		// is user to be followed is public , create follow

		if !userToBeFollowed.IsPublic {
			writeJSONError(w, "user to be followed should be public , otherwise a request has to be sent first", http.StatusBadRequest)
			return
		}

		follow, err := h.storage.CreateFollow(user.Id, userToBeFollowed.Id)
		if err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		type Response struct {
			Success bool           `json:"success"`
			Message string         `json:"message"`
			Follow  storage.Follow `json:"follow"`
		}

		if err := writeJSON(w, Response{Success: true, Message: "followed user", Follow: *follow}, http.StatusCreated); err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
		}
	} else {
		if err := h.storage.RemoveFollow(user.Id, userToBeFollowed.Id); err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
		type Response struct {
			Success bool   `json:"success"`
			Message string `json:"message"`
		}

		if err := writeJSON(w, Response{Success: true, Message: "unfollowed user"}, http.StatusOK); err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}
}

func (h *Handler) AcceptFollowRequestHandler(w http.ResponseWriter, r *http.Request) {
	userId, ok := r.Context().Value(AuthUserId).(int)
	if !ok {
		log.Println("user id not of type int when asserting as int")
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	requestReceiver, err := h.storage.GetUserById(userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "user not found", http.StatusBadRequest)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	requestSenderId, err := strconv.Atoi(chi.URLParam(r, "userId"))
	if err != nil {
		writeJSONError(w, "invalid request param", http.StatusBadRequest)
		return
	}

	requestSender, err := h.storage.GetUserById(requestSenderId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "request sender user not found", http.StatusBadRequest)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	// check if follow request from sender exists for receiver
	followRequest, err := h.storage.GetFollowRequest(requestSender.Id, requestReceiver.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "follow request not found", http.StatusBadRequest)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	// so what is accepting a follow request
	// accepting is basically now the sender follows the receiver
	// so create a follow from sender to receiver
	// delete the request after follow is done

	follow, err := h.storage.AcceptFollowRequest(followRequest.RequestSenderId, followRequest.RequestReceiverId)
	if err != nil {
		log.Printf("failed to accept follow request :- %v\n", err.Error())
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	type Response struct {
		Success bool           `json:"success"`
		Message string         `json:"message"`
		Follow  storage.Follow `json:"follow"`
	}

	if err := writeJSON(w, Response{Success: true, Message: "accepted follow request", Follow: *follow}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
	}
}

func (h *Handler) GetUserPostsHandler(w http.ResponseWriter, r *http.Request) {

	authUserId, ok := r.Context().Value(AuthUserId).(int)
	if !ok {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	authUser, err := h.storage.GetUserById(authUserId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "user not found", http.StatusBadRequest)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return

		}
	}

	userId, err := strconv.Atoi(chi.URLParam(r, "userId"))
	if err != nil {
		writeJSONError(w, "invalid request param userId", http.StatusBadRequest)
		return
	}

	user, err := h.storage.GetUserById(userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "user not found", http.StatusBadRequest)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	page, err := strconv.Atoi(r.URL.Query().Get("page"))

	if err != nil {
		writeJSONError(w, "invalid query param page", http.StatusBadRequest)
		return
	}

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		writeJSONError(w, "invalid query param limit", http.StatusBadRequest)
		return
	}

	skip := page*limit - limit

	follow, err := h.storage.GetFollow(authUser.Id, user.Id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if !user.IsPublic && follow == nil && authUser.Id != user.Id {
		writeJSONError(w, "user is not public and not followed by auth user", http.StatusUnauthorized)
		return
	}

	posts, err := h.storage.GetPostsByUserId(user.Id, skip, limit)
	if err != nil {
		return
	}

	totalPostsCount, err := h.storage.GetPostsCountByUser(user.Id)
	if err != nil {
		return
	}

	noOfPages := math.Ceil(float64(totalPostsCount) / float64(limit))

	type Response struct {
		Success   bool                       `json:"success"`
		Posts     []storage.PostWithMetaData `json:"posts"`
		NoOfPages int                        `json:"noOfPages"`
	}

	if err := writeJSON(w, Response{Success: true, Posts: posts, NoOfPages: int(noOfPages)}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
	}
}

func (h *Handler) GetUserLikedPostsHandler(w http.ResponseWriter, r *http.Request) {

	authUserId, ok := r.Context().Value(AuthUserId).(int)
	if !ok {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	authUser, err := h.storage.GetUserById(authUserId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "user not found", http.StatusBadRequest)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return

		}
	}

	userId, err := strconv.Atoi(chi.URLParam(r, "userId"))
	if err != nil {
		writeJSONError(w, "invalid request param userId", http.StatusBadRequest)
		return
	}

	user, err := h.storage.GetUserById(userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "user not found", http.StatusBadRequest)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil {
		writeJSONError(w, "invalid query param page", http.StatusBadRequest)
		return
	}

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		writeJSONError(w, "invalid query param limit", http.StatusBadRequest)
		return
	}

	skip := page*limit - limit

	follow, err := h.storage.GetFollow(authUser.Id, user.Id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if !user.IsPublic && follow == nil && authUserId != user.Id {
		writeJSONError(w, "user is not public and not followed by auth user", http.StatusUnauthorized)
		return
	}

	likedPosts, err := h.storage.GetLikedPostsByUser(user.Id, skip, limit)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return

	}

	totalLikedPosts, err := h.storage.GetLikedPostsByUserCount(user.Id)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	noOfPages := math.Ceil(float64(totalLikedPosts) / float64(limit))

	type Response struct {
		Success    bool                       `json:"success"`
		LikedPosts []storage.PostWithMetaData `json:"liked_posts"`
		NoOfPages  int                        `json:"noOfPages"`
	}

	if err := writeJSON(w, Response{Success: true, LikedPosts: likedPosts, NoOfPages: int(noOfPages)}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
	}
}

func (h *Handler) GetUserBookmarkedPostsHandler(w http.ResponseWriter, r *http.Request) {
	authUserId, ok := r.Context().Value(AuthUserId).(int)
	if !ok {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	authUser, err := h.storage.GetUserById(authUserId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "user not found", http.StatusBadRequest)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return

		}
	}

	userId, err := strconv.Atoi(chi.URLParam(r, "userId"))
	if err != nil {
		writeJSONError(w, "invalid request param userId", http.StatusBadRequest)
		return
	}

	user, err := h.storage.GetUserById(userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "user not found", http.StatusBadRequest)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil {
		writeJSONError(w, "invalid query param page", http.StatusBadRequest)
		return
	}

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		writeJSONError(w, "invalid query param limit", http.StatusBadRequest)
		return
	}

	skip := page*limit - limit

	follow, err := h.storage.GetFollow(authUser.Id, user.Id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if !user.IsPublic && follow == nil && authUserId != user.Id {
		writeJSONError(w, "user is not public and not followed by auth user", http.StatusUnauthorized)
		return
	}

	posts, err := h.storage.GetBookmarkedPostsByUser(user.Id, skip, limit)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return

	}

	totalBookmarkedPosts, err := h.storage.GetBookmarkedPostsByUserCount(user.Id)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return

	}

	noOfPages := math.Ceil(float64(totalBookmarkedPosts) / float64(limit))

	type Response struct {
		Success         bool                       `json:"success"`
		BookmarkedPosts []storage.PostWithMetaData `json:"bookmarked_posts"`
		NoOfPages       int                        `json:"noOfPages"`
	}

	if err := writeJSON(w, Response{Success: true, BookmarkedPosts: posts, NoOfPages: int(noOfPages)}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
	}
}

func (h *Handler) GetUserFollowersHandler(w http.ResponseWriter, r *http.Request) {
	authUserId, ok := r.Context().Value(AuthUserId).(int)
	if !ok {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	authUser, err := h.storage.GetUserById(authUserId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "user not found", http.StatusBadRequest)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return

		}
	}

	userId, err := strconv.Atoi(chi.URLParam(r, "userId"))
	if err != nil {
		writeJSONError(w, "invalid request param userId", http.StatusBadRequest)
		return
	}

	user, err := h.storage.GetUserById(userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "user not found", http.StatusBadRequest)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil {
		writeJSONError(w, "invalid query param page", http.StatusBadRequest)
		return
	}

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		writeJSONError(w, "invalid query param limit", http.StatusBadRequest)
		return
	}

	skip := page*limit - limit

	follow, err := h.storage.GetFollow(authUser.Id, user.Id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if !user.IsPublic && follow == nil && authUser.Id != user.Id {
		writeJSONError(w, "cannot view user's followers , user is private and auth user is not following the user", http.StatusBadRequest)
		return
	}

	followers, err := h.storage.GetFollowers(user.Id, skip, limit)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	totalFollowersCount, err := h.storage.GetFollowersCount(user.Id)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	noOfPages := math.Ceil(float64(totalFollowersCount) / float64(limit))

	type Response struct {
		Success   bool           `json:"success"`
		Followers []storage.User `json:"followers"`
		NoOfPages int            `json:"noOfPages"`
	}

	if err := writeJSON(w, Response{Success: true, Followers: followers, NoOfPages: int(noOfPages)}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) GetUserFollowingsHandler(w http.ResponseWriter, r *http.Request) {
	authUserId, ok := r.Context().Value(AuthUserId).(int)
	if !ok {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	authUser, err := h.storage.GetUserById(authUserId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "user not found", http.StatusBadRequest)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return

		}
	}

	userId, err := strconv.Atoi(chi.URLParam(r, "userId"))
	if err != nil {
		writeJSONError(w, "invalid request param userId", http.StatusBadRequest)
		return
	}

	user, err := h.storage.GetUserById(userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "user not found", http.StatusBadRequest)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil {
		writeJSONError(w, "invalid query param page", http.StatusBadRequest)
		return
	}

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		writeJSONError(w, "invalid query param limit", http.StatusBadRequest)
		return
	}

	skip := page*limit - limit

	follow, err := h.storage.GetFollow(authUser.Id, user.Id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if !user.IsPublic && follow == nil && authUser.Id != user.Id {
		writeJSONError(w, "cannot view user's followers , user is private and auth user is not following the user", http.StatusBadRequest)
		return
	}

	followings, err := h.storage.GetFollowings(user.Id, skip, limit)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	totalFollowingsCount, err := h.storage.GetFollowingsCount(user.Id)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	noOfPages := math.Ceil(float64(totalFollowingsCount) / float64(limit))

	type Response struct {
		Success    bool           `json:"success"`
		Followings []storage.User `json:"followings"`
		NoOfPages  int            `json:"noOfPages"`
	}

	if err := writeJSON(w, Response{Success: true, Followings: followings, NoOfPages: int(noOfPages)}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) GetUserProfileHandler(w http.ResponseWriter, r *http.Request) {

	userId, err := strconv.Atoi(chi.URLParam(r, "userId"))
	if err != nil {
		writeJSONError(w, "invalid request param userId", http.StatusBadRequest)
		return
	}

	user, err := h.storage.GetUserById(userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "user not found", http.StatusBadRequest)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	type UserProfileData struct {
		storage.User
		NoOfPosts       int `json:"no_of_posts"`
		FollowersCount  int `json:"followers_count"`
		FollowingsCount int `json:"followings_count"`
	}

	followersCount, err := h.storage.GetFollowersCount(user.Id)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	followingsCount, err := h.storage.GetFollowingsCount(user.Id)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	noOfPosts, err := h.storage.GetPostsCountByUser(user.Id)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	userProfileData := UserProfileData{User: *user, NoOfPosts: noOfPosts, FollowersCount: followersCount, FollowingsCount: followingsCount}

	type Response struct {
		Success bool            `json:"success"`
		Profile UserProfileData `json:"profile"`
	}

	if err := writeJSON(w, Response{Success: true, Profile: userProfileData}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

type UpdateUserRequest struct {
	Username string `json:"username"`
	ImageUrl string `json:"image_url"`
	Bio      string `json:"bio"`
	Location string `json:"location"`
	IsPublic bool   `json:"is_public"`
}

func (h *Handler) UpdateUserHandler(w http.ResponseWriter, r *http.Request) {

	userId, ok := r.Context().Value(AuthUserId).(int)
	if !ok {
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
			return
		}
	}

	var updateUserPayload UpdateUserRequest

	if err := json.NewDecoder(r.Body).Decode(&updateUserPayload); err != nil {
		writeJSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	newUsername := strings.TrimSpace(updateUserPayload.Username)
	newBio := strings.TrimSpace(updateUserPayload.Bio)
	newLocation := strings.TrimSpace(updateUserPayload.Location)
	newImageUrl := strings.TrimSpace(updateUserPayload.ImageUrl)
	isUserPublic := updateUserPayload.IsPublic

	if newUsername == "" {
		writeJSONError(w, "username cannot be empty", http.StatusBadRequest)
		return
	}

	if utf8.RuneCountInString(newUsername) < 3 {
		writeJSONError(w, "new username cannot have less than 3 characters", http.StatusBadRequest)
		return
	}
	// check if a user already exists with this username
	existingUser, err := h.storage.GetUserByUsername(newUsername)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if existingUser != nil && existingUser.Username != user.Username {
		writeJSONError(w, "username already taken", http.StatusBadRequest)
		return
	}

	updatedUser, err := h.storage.UpdateUser(user.Id, newUsername, newImageUrl, newBio, newLocation, isUserPublic)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return

	}

	type Response struct {
		Success     bool         `json:"success"`
		Message     string       `json:"message"`
		UpdatedUser storage.User `json:"updated_user"`
	}

	if err := writeJSON(w, Response{Success: true, Message: "updated user successfully", UpdatedUser: *updatedUser}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
	}
}

func (h *Handler) GetFollowRequestsSentHandler(w http.ResponseWriter, r *http.Request) {
	userId, ok := r.Context().Value(AuthUserId).(int)
	if !ok {
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
			return
		}
	}

	followRequests, err := h.storage.GetFollowRequestsSentByUser(user.Id)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	type Response struct {
		Success        bool                    `json:"success"`
		FollowRequests []storage.FollowRequest `json:"follow_requests"`
	}

	if err := writeJSON(w, Response{Success: true, FollowRequests: followRequests}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) GetFollowingsHandler(w http.ResponseWriter, r *http.Request) {
	userId, ok := r.Context().Value(AuthUserId).(int)
	if !ok {
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
			return
		}
	}

	followings, err := h.storage.GetFollowingsByUser(user.Id)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	type Response struct {
		Success    bool             `json:"success"`
		Followings []storage.Follow `json:"followings"`
	}

	if err := writeJSON(w, Response{Success: true, Followings: followings}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) GetNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	userId, ok := r.Context().Value(AuthUserId).(int)
	if !ok {
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
			return
		}
	}

	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil {
		writeJSONError(w, "invalid query params page", http.StatusBadRequest)
		return
	}

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		writeJSONError(w, "invalid query params limit", http.StatusBadRequest)
		return
	}

	skip := page*limit - limit

	notifications, err := h.storage.GetNotificationsByUserId(user.Id, skip, limit)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	totalNotificationsCount, err := h.storage.GetNotificationsByUserIdCount(user.Id)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	noOfPages := math.Ceil(float64(totalNotificationsCount) / float64(limit))

	type Response struct {
		Success       bool                            `json:"success"`
		Notifications []storage.NotificationWithActor `json:"notifications"`
		NoOfPages     int                             `json:"noOfPages"`
	}

	if err := writeJSON(w, Response{Success: true, Notifications: notifications, NoOfPages: int(noOfPages)}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) GetRequestsReceivedHandler(w http.ResponseWriter, r *http.Request) {
	userId, ok := r.Context().Value(AuthUserId).(int)
	if !ok {
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
			return
		}
	}

	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil {
		writeJSONError(w, "invalid query params page", http.StatusBadRequest)
		return
	}

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		writeJSONError(w, "invalid query params limit", http.StatusBadRequest)
		return
	}

	skip := page*limit - limit

	followRequestsReceived, err := h.storage.GetFollowRequestsReceivedByUser(user.Id, skip, limit)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	totalRequestsReceivedCount, err := h.storage.GetFollowRequestsReceivedByUserCount(user.Id)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	noOfPages := math.Ceil(float64(totalRequestsReceivedCount) / float64(limit))

	type Response struct {
		Success                bool                              `json:"success"`
		FollowRequestsReceived []storage.FollowRequestWithSender `json:"follow_requests_received"`
		NoOfPages              int                               `json:"noOfPages"`
	}

	if err := writeJSON(w, Response{Success: true, FollowRequestsReceived: followRequestsReceived, NoOfPages: int(noOfPages)}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
}
