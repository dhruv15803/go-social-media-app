package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/dhruv15803/social-media-app/storage"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) GetPostBookmarksHandler(w http.ResponseWriter, r *http.Request) {

	postId, err := strconv.Atoi(chi.URLParam(r, "postId"))
	if err != nil {
		writeJSONError(w, "invalid request params postId", http.StatusBadRequest)
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

	// get post bookmarks
	bookmarks, err := h.storage.GetBookmarksByPostId(post.Id)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	type Response struct {
		Success   bool               `json:"success"`
		Bookmarks []storage.Bookmark `json:"bookmarks"`
	}

	if err := writeJSON(w, Response{Success: true, Bookmarks: bookmarks}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
}
