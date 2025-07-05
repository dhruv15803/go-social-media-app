package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/dhruv15803/social-media-app/storage"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) GetPostLikesHandler(w http.ResponseWriter, r *http.Request) {
	postId, err := strconv.Atoi(chi.URLParam(r, "postId"))
	if err != nil {
		writeJSONError(w, "invalid request parameter", http.StatusBadRequest)
		return
	}

	post, err := h.storage.GetPostById(postId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "invalid post id", http.StatusBadRequest)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	// get likes for this post i.e liked_post_id=post.id
	likes, err := h.storage.GetPostLikes(post.Id)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	type Response struct {
		Success bool           `json:"success"`
		Likes   []storage.Like `json:"likes"`
	}

	if err := writeJSON(w, Response{Success: true, Likes: likes}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
}
