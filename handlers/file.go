package handlers

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

func (h *Handler) UploadFileHandler(w http.ResponseWriter, r *http.Request) {

	file, fileHeader, err := r.FormFile("imageFile")

	if err != nil {
		log.Printf("failed to access imageFile from multipart/form-data :- %v\n", err.Error())
		writeJSONError(w, "image file not found", http.StatusBadRequest)
		return
	}

	dst, err := os.Create("./uploads/" + fileHeader.Filename)
	if err != nil {
		log.Printf("failed to create destination for file :- %v\n", err.Error())
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	_, err = io.Copy(dst, file)
	if err != nil {
		log.Printf("failed to copy imageFile content to destination file :- %v\n", err.Error())
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	uploadedFile, err := os.Open(dst.Name())
	if err != nil {
		log.Printf("failed to open uploaded file at destination :- %v\n", err.Error())
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// upload file to cloudinary
	result, err := h.cld.Upload.Upload(context.Background(), uploadedFile, uploader.UploadParams{})
	if err != nil {
		log.Printf("failed to upload to cloudinary :- %v\n", err.Error())
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	uploadedFile.Close()
	dst.Close()

	// remove uploaded file from server
	if err = os.Remove(uploadedFile.Name()); err != nil {
		log.Printf("failed to remove uploaded file from server after upload :- %v\n", err.Error())
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	type Response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Url     string `json:"url"`
	}

	if err := writeJSON(w, Response{Success: true, Message: "file uploaded successfully", Url: result.SecureURL}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
}
