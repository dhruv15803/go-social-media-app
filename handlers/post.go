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

	"github.com/dhruv15803/social-media-app/storage"
	"github.com/go-chi/chi/v5"
)

type CreatePostRequest struct {
	PostContent   string   `json:"post_content"`
	PostImageUrls []string `json:"post_image_urls"`
}

type CreateChildPostRequest struct {
	PostContent   string   `json:"post_content"`
	PostImageUrls []string `json:"post_image_urls"`
}

const (
	likesCountWt     float64 = 0.7
	commentsCountWt  float64 = 0.8
	bookmarksCountWt float64 = 0.5
)

func (h *Handler) GetPostsHandler(w http.ResponseWriter, r *http.Request) {

	userId, ok := r.Context().Value(AuthUserId).(int)
	if !ok {
		log.Println("AuthUserId from context is not an integer")
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	user, err := h.storage.GetUserById(userId)
	if err != nil {
		writeJSONError(w, "authenticated user not found", http.StatusBadRequest)
		return
	}

	pageNum, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil {
		writeJSONError(w, "invaild query params page", http.StatusBadRequest)
		return
	}

	limitNum, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		writeJSONError(w, "invalid query params limit", http.StatusBadRequest)
		return
	}

	skip := pageNum*limitNum - limitNum
	posts, err := h.storage.GetUserPostFeed(skip, limitNum, user.Id, likesCountWt, commentsCountWt, bookmarksCountWt)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	totalPostsCount, err := h.storage.GetUserPostFeedCount(user.Id)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	noOfPages := math.Ceil(float64(totalPostsCount) / float64(limitNum))

	type Response struct {
		Success   bool                       `json:"success"`
		Posts     []storage.PostWithMetaData `json:"posts"`
		NoOfPages int                        `json:"noOfPages"`
	}

	if err := writeJSON(w, Response{Success: true, Posts: posts, NoOfPages: int(noOfPages)}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) GetPublicPostsHandler(w http.ResponseWriter, r *http.Request) {
	// this /post/posts handler is a unauthenticated handler as to view
	// posts on a social media website , there is no authentication required
	// so this endpoint returns posts according to their activity
	// activity -> no of likes, comments , bookmarks
	// activityScore := (no Of Likes + no Of Comments + noOfBookmarks)
	// so return posts from highest activity score to lowest
	// but if the high activity score posts are too old (past a certain threshold)
	// prioritize latest posts

	pageNum, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil {
		writeJSONError(w, "invalid query param page", http.StatusBadRequest)
		return
	}
	limitNum, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		writeJSONError(w, "invalid query param limit", http.StatusBadRequest)
		return
	}

	skip := pageNum*limitNum - limitNum

	// top level posts

	posts, err := h.storage.GetPublicPosts(skip, limitNum, likesCountWt, commentsCountWt, bookmarksCountWt)
	if err != nil {
		log.Printf("failed to fetch posts :- %v\n", err.Error())
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	totalTopLevelPostsCount, err := h.storage.GetPublicPostsCount()
	if err != nil {
		log.Printf("failed to fetch top level posts count :- %v\n", err.Error())
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	noOfPages := math.Ceil(float64(totalTopLevelPostsCount) / float64(limitNum))

	type Response struct {
		Success   bool                       `json:"success"`
		Posts     []storage.PostWithMetaData `json:"posts"`
		NoOfPages int                        `json:"noOfPages"`
	}

	if err := writeJSON(w, Response{Success: true, Posts: posts, NoOfPages: int(noOfPages)}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

// this is an endpoint for creating a top level post ,(no comment posts)
func (h *Handler) CreatePostHandler(w http.ResponseWriter, r *http.Request) {
	userId, ok := r.Context().Value(AuthUserId).(int)
	if !ok {
		log.Println("AuthUserId from context is not an integer")
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	user, err := h.storage.GetUserById(userId)
	if err != nil {
		writeJSONError(w, "authenticated user not found", http.StatusBadRequest)
		return
	}

	var createPostPayload CreatePostRequest
	isPostWithImages := false

	if err := json.NewDecoder(r.Body).Decode(&createPostPayload); err != nil {
		writeJSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	postContent := strings.TrimSpace(createPostPayload.PostContent)
	postImageUrls := createPostPayload.PostImageUrls

	if postContent == "" {
		writeJSONError(w, "post content is required", http.StatusBadRequest)
		return
	}

	if len(postImageUrls) != 0 {
		isPostWithImages = true
	}

	if isPostWithImages {

		newPost, err := h.storage.CreatePostWithImages(postContent, postImageUrls, user.Id)
		if err != nil {
			log.Printf("failed to create post with images :- %v\n", err.Error())
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		type Response struct {
			Success bool                          `json:"success"`
			Message string                        `json:"message"`
			Post    storage.PostWithUserAndImages `json:"post"`
		}

		if err := writeJSON(w, Response{Success: true, Message: "created post successfully", Post: *newPost}, http.StatusCreated); err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	} else {

		newPost, err := h.storage.CreatePost(postContent, user.Id)
		if err != nil {
			log.Printf("failed to create post :- %v\n", err.Error())
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		type Response struct {
			Success bool                 `json:"success"`
			Message string               `json:"message"`
			Post    storage.PostWithUser `json:"post"`
		}

		if err := writeJSON(w, Response{Success: true, Message: "created post successfully", Post: *newPost}, http.StatusCreated); err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}
}

func (h *Handler) CreateChildPostHandler(w http.ResponseWriter, r *http.Request) {

	userId, ok := r.Context().Value(AuthUserId).(int)
	if !ok {
		log.Println("AuthUserId from context is not an integer")
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	user, err := h.storage.GetUserById(userId)
	if err != nil {
		writeJSONError(w, "authenticated user not found", http.StatusBadRequest)
		return
	}

	parentPostId, err := strconv.Atoi(chi.URLParam(r, "parentPostId"))
	if err != nil {
		writeJSONError(w, "invalid request param", http.StatusBadRequest)
		return
	}

	parentPost, err := h.storage.GetPostById(parentPostId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "parent post not found", http.StatusBadRequest)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	parentPostOwnerId := parentPost.UserId

	var createChildPostPayload CreateChildPostRequest

	if err := json.NewDecoder(r.Body).Decode(&createChildPostPayload); err != nil {
		writeJSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	postContent := strings.TrimSpace(createChildPostPayload.PostContent)
	postImageUrls := createChildPostPayload.PostImageUrls
	isPostWithImages := false

	if postContent == "" {
		writeJSONError(w, "post content is required", http.StatusBadRequest)
		return
	}

	if len(postImageUrls) != 0 {
		isPostWithImages = true
	}

	if isPostWithImages {
		// create child post with images

		post, err := h.storage.CreateChildPostWithImages(postContent, postImageUrls, user.Id, parentPost.Id)
		if err != nil {
			log.Printf("failed to create child post with images :- %v\n", err.Error())
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if parentPostOwnerId != user.Id {
			maxRetries := 3
			if ok := h.sendNotification(parentPost.Id, user.Id, "comment", parentPost.Id, maxRetries); !ok {
				writeJSONError(w, "internal server error", http.StatusInternalServerError)
				return
			}
		}

		type Response struct {
			Success bool                          `json:"success"`
			Message string                        `json:"message"`
			Post    storage.PostWithUserAndImages `json:"post"`
		}

		if err := writeJSON(w, Response{Success: true, Message: "post created successfully", Post: *post}, http.StatusCreated); err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	} else {

		post, err := h.storage.CreateChildPost(postContent, user.Id, parentPost.Id)
		if err != nil {
			log.Printf("failed to create child post :- %v\n", err.Error())
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if parentPostOwnerId != user.Id {
			maxRetries := 3
			if ok := h.sendNotification(parentPostOwnerId, user.Id, "comment", parentPost.Id, maxRetries); !ok {
				writeJSONError(w, "internal server error", http.StatusInternalServerError)
				return
			}
		}

		type Response struct {
			Success bool                 `json:"success"`
			Message string               `json:"message"`
			Post    storage.PostWithUser `json:"post"`
		}

		if err := writeJSON(w, Response{Success: true, Message: "post created successfully", Post: *post}, http.StatusCreated); err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}
}

func (h *Handler) DeletePostHandler(w http.ResponseWriter, r *http.Request) {

	userId, ok := r.Context().Value(AuthUserId).(int)
	if !ok {
		log.Println("AuthUserId from context is not an integer")
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	user, err := h.storage.GetUserById(userId)
	if err != nil {
		writeJSONError(w, "authenticated user not found", http.StatusBadRequest)
		return
	}

	postId, err := strconv.Atoi(chi.URLParam(r, "postId"))
	if err != nil {
		writeJSONError(w, "invalid request parameter", http.StatusBadRequest)
		return
	}

	post, err := h.storage.GetPostById(postId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "post not found", http.StatusBadRequest)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	if post.UserId != user.Id {
		writeJSONError(w, "unauthorized to delete post", http.StatusUnauthorized)
		return
	}

	if err = h.storage.DeletePostById(post.Id); err != nil {
		log.Printf("failed to delete post with id %v , error - %v", post.Id, err.Error())
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	type Response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	if err := writeJSON(w, Response{Success: true, Message: "post deleted successfully"}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) LikePostHandler(w http.ResponseWriter, r *http.Request) {
	userId, ok := r.Context().Value(AuthUserId).(int)
	if !ok {
		log.Println("AuthUserId from context is not an integer")
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	user, err := h.storage.GetUserById(userId)
	if err != nil {
		writeJSONError(w, "authenticated user not found", http.StatusBadRequest)
		return
	}

	postId, err := strconv.Atoi(chi.URLParam(r, "postId"))
	if err != nil {
		writeJSONError(w, "invalid request param", http.StatusBadRequest)
		return
	}

	post, err := h.storage.GetPostById(postId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "post not found", http.StatusBadRequest)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	// here the auth user is trying to like a post
	// so check if like by the user on this post already exists

	existingLike, err := h.storage.GetLike(user.Id, post.Id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Printf("failed operation to get like by user %d on post %d , err :- %v", user.Id, post.Id, err.Error())
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if existingLike == nil {
		// create like
		like, err := h.storage.CreateLike(user.Id, post.Id)
		if err != nil {
			log.Printf("failed to create like :- %v\n", err.Error())
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
		// post liked by user , create 'liked' notification for post owner
		postOwnerId := post.UserId

		if postOwnerId != user.Id {
			// only send like notification if somebody else's post

			// check if  user had already liked before this and then unliked then liking again
			// so there will be an existing notification entry with actor_id=user.Id,post_id=post.Id,type="like"

			existingLikeNotifications, err := h.storage.GetNotificationsByActorIdAndPostId(user.Id, post.Id, "like")
			if err != nil {
				writeJSONError(w, "internal server error", http.StatusInternalServerError)
				return
			}

			if len(existingLikeNotifications) != 0 {
				// update the notification_created_at field

				_, err := h.storage.UpdateNotificationByActorIdAndPostId(user.Id, post.Id, "like")
				if err != nil {
					log.Printf("failed to update notification created_at :- %v\n", err.Error())
					writeJSONError(w, "internal server error", http.StatusInternalServerError)
					return
				}

			} else {
				maxNotificationRetries := 3

				if ok := h.sendNotification(postOwnerId, user.Id, "like", post.Id, maxNotificationRetries); !ok {
					log.Println("failed to create like notification")
					writeJSONError(w, "internal server error", http.StatusInternalServerError)
					return
				}
			}
		}

		type Response struct {
			Success bool         `json:"success"`
			Message string       `json:"message"`
			Like    storage.Like `json:"like"`
		}

		if err := writeJSON(w, Response{Success: true, Message: "liked post", Like: *like}, http.StatusCreated); err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
		}

	} else {
		if err := h.storage.RemoveLike(existingLike.LikedById, existingLike.LikedPostId); err != nil {
			log.Printf("failed to delete existing like :- %v\n", err.Error())
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		type Response struct {
			Success bool   `json:"success"`
			Message string `json:"message"`
		}

		if err := writeJSON(w, Response{Success: true, Message: "unliked post"}, http.StatusOK); err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
		}
	}
}

func (h *Handler) BookmarkPostHandler(w http.ResponseWriter, r *http.Request) {

	userId, ok := r.Context().Value(AuthUserId).(int)
	if !ok {
		log.Println("AuthUserId from context is not an integer")
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	user, err := h.storage.GetUserById(userId)
	if err != nil {
		writeJSONError(w, "authenticated user not found", http.StatusBadRequest)
		return
	}

	postId, err := strconv.Atoi(chi.URLParam(r, "postId"))
	if err != nil {
		writeJSONError(w, "invalid request param", http.StatusBadRequest)
		return
	}

	post, err := h.storage.GetPostById(postId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "post not found", http.StatusBadRequest)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	// check if bookmarked already by user
	existingBookmark, err := h.storage.GetBookmark(user.Id, postId)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if existingBookmark == nil {

		// create bookmark
		bookmark, err := h.storage.CreateBookmark(user.Id, post.Id)
		if err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		type Response struct {
			Success  bool             `json:"success"`
			Message  string           `json:"message"`
			Bookmark storage.Bookmark `json:"bookmark"`
		}

		if err := writeJSON(w, Response{Success: true, Message: "bookmarked post", Bookmark: *bookmark}, http.StatusCreated); err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)

		}

	} else {
		// remove bookmark
		if err := h.storage.RemoveBookmark(existingBookmark.BookmarkedById, existingBookmark.BookmarkedPostId); err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		type Response struct {
			Success bool   `json:"success"`
			Message string `json:"message"`
		}

		if err := writeJSON(w, Response{Success: true, Message: "removed bookmark"}, http.StatusOK); err != nil {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
		}
	}
}

func (h Handler) GetMyPostsHandler(w http.ResponseWriter, r *http.Request) {

	userId, ok := r.Context().Value(AuthUserId).(int)
	if !ok {
		log.Println("AuthUserId from context is not an integer")
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	user, err := h.storage.GetUserById(userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "authenticated user not found", http.StatusBadRequest)
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

	posts, err := h.storage.GetPostsByUserId(user.Id, skip, limit)
	if err != nil {
		log.Printf("failed to fetch user's posts :- %v\n", err.Error())
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	usersTopLevelPostsCount, err := h.storage.GetPostsCountByUser(user.Id)
	if err != nil {
		log.Printf("failed to fetch user's no of top level posts :- %v\n", err.Error())
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	noOfPages := math.Ceil(float64(usersTopLevelPostsCount) / float64(limit))

	type Response struct {
		Success   bool                       `json:"success"`
		Posts     []storage.PostWithMetaData `json:"posts"`
		NoOfPages int                        `json:"noOfPages"`
	}

	if err := writeJSON(w, Response{Success: true, Posts: posts, NoOfPages: int(noOfPages)}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) GetPostCommentsHandler(w http.ResponseWriter, r *http.Request) {

	postId, err := strconv.Atoi(chi.URLParam(r, "postId"))
	if err != nil {
		writeJSONError(w, "invalid request param postId", http.StatusBadRequest)
		return
	}

	post, err := h.storage.GetPostById(postId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "post not found", http.StatusBadRequest)
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

	// get  comments for this post i.e posts where parent_post_id=post.Id

	comments, err := h.storage.GetPostComments(post.Id, skip, limit)
	if err != nil {
		log.Printf("failed to fetch post comments :- %v\n", err.Error())
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	totalPostCommentsCount, err := h.storage.GetPostCommentsCount(post.Id)
	if err != nil {
		log.Printf("failed to fetch post comments count :- %v\n", err.Error())
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	noOfPages := math.Ceil(float64(totalPostCommentsCount) / float64(limit))

	type Response struct {
		Success   bool                       `json:"success"`
		Comments  []storage.PostWithMetaData `json:"comments"`
		NoOfPages int                        `json:"noOfPages"`
	}

	if err := writeJSON(w, Response{Success: true, Comments: comments, NoOfPages: int(noOfPages)}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
	}
}

func (h *Handler) GetMyLikedPostsHandler(w http.ResponseWriter, r *http.Request) {

	userId, ok := r.Context().Value(AuthUserId).(int)
	if !ok {
		log.Println("AuthUserId from context is not an integer")
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	user, err := h.storage.GetUserById(userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "authenticated user not found", http.StatusBadRequest)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	//get user's liked posts
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

	likedPosts, err := h.storage.GetLikedPostsByUser(user.Id, skip, limit)
	if err != nil {
		log.Printf("failed to fetch liked posts :- %v\n", err.Error())
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	totalLikedPosts, err := h.storage.GetLikedPostsByUserCount(user.Id)
	if err != nil {
		log.Printf("failed to fetch total liked posts by user :- %v\n", err.Error())
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

func (h *Handler) GetPostHandler(w http.ResponseWriter, r *http.Request) {
	postId, err := strconv.Atoi(chi.URLParam(r, "postId"))
	if err != nil {
		writeJSONError(w, "invalid request param postId", http.StatusBadRequest)
		return
	}

	post, err := h.storage.GetPostById(postId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "post not found", http.StatusBadRequest)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	type Response struct {
		Success bool         `json:"success"`
		Post    storage.Post `json:"post"`
	}

	if err := writeJSON(w, Response{Success: true, Post: *post}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
	}
}

func (h *Handler) GetPostWithMetaDataHandler(w http.ResponseWriter, r *http.Request) {

	postId, err := strconv.Atoi(chi.URLParam(r, "postId"))
	if err != nil {
		writeJSONError(w, "invalid request param postId", http.StatusBadRequest)
		return
	}

	post, err := h.storage.GetPostById(postId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "post not found", http.StatusBadRequest)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	postWithMetaData, err := h.storage.GetPostWithMetaDataById(post.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "post not found", http.StatusBadRequest)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	type Response struct {
		Success bool                     `json:"success"`
		Post    storage.PostWithMetaData `json:"post"`
	}

	if err := writeJSON(w, Response{Success: true, Post: *postWithMetaData}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
	}
}

func (h *Handler) sendNotification(userId int, actorId int, notificationType storage.NotificationType, postId int, maxRetries int) bool {

	isNotificationSuccessful := false

	for i := 0; i < maxRetries; i++ {
		_, err := h.storage.CreateNotification(userId, actorId, postId, notificationType)
		if err != nil {
			log.Printf(err.Error())
			continue
		}
		isNotificationSuccessful = true
		break
	}

	return isNotificationSuccessful
}

func (h *Handler) GetPostLikedUsersHandler(w http.ResponseWriter, r *http.Request) {
	postId, err := strconv.Atoi(chi.URLParam(r, "postId"))
	if err != nil {
		writeJSONError(w, "invalid request param postId", http.StatusBadRequest)
		return
	}

	post, err := h.storage.GetPostById(postId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "post not found", http.StatusBadRequest)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	pageNum, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil {
		writeJSONError(w, "invalid query param page", http.StatusBadRequest)
		return
	}

	limitNum, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		writeJSONError(w, "invalid query param limit", http.StatusBadRequest)
		return
	}

	skip := pageNum*limitNum - limitNum

	users, err := h.storage.GetPostLikedUsers(post.Id, skip, limitNum)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	totalPostLikesCount, err := h.storage.GetPostLikedUsersCount(post.Id)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	noOfPages := int(math.Ceil(float64(totalPostLikesCount) / float64(limitNum)))

	type Response struct {
		Success   bool           `json:"success"`
		Users     []storage.User `json:"users"`
		NoOfPages int            `json:"noOfPages"`
	}

	if err := writeJSON(w, Response{Success: true, Users: users, NoOfPages: noOfPages}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
	}
}
