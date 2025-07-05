package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/dhruv15803/social-media-app/storage"
)

type Handler struct {
	storage storage.Storage
	cld     *cloudinary.Cloudinary
}

func NewHandler(storage storage.Storage, cld *cloudinary.Cloudinary) *Handler {
	return &Handler{
		storage: storage,
		cld:     cld,
	}
}

func (h *Handler) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"message": "server health OK"}); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}
