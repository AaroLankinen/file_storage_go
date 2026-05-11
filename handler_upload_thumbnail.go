package main

import (
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// Parse the multipart form data with a maximum memory of 10 MB
	maxMemory := int64(10 << 20) // 10 MB
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse multipart form", err)
		return
	}

	// Get the file from the form data
	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't get thumbnail from form data", err)
		return
	}
	defer file.Close()

	// Read the file into a byte slice
	buf := make([]byte, header.Size)
	_, err = file.Read(buf)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't read thumbnail file", err)
		return
	}

	dataURL := fmt.Sprintf("data:%s;base64,%s", header.Header.Get("Content-Type"), base64.StdEncoding.EncodeToString(buf))

	// Check if the video exists and belongs to the user
	metadata, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Video not found", err)
		return
	}

	if metadata.UserID != userID {
		respondWithError(w, http.StatusForbidden, "You don't have permission to upload a thumbnail for this video", nil)
		return
	}

	metadata.ThumbnailURL = &dataURL
	err = cfg.db.UpdateVideo(metadata)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video metadata with thumbnail URL", err)
		return
	}

	respondWithJSON(w, http.StatusOK, database.Video{
		ID:           metadata.ID,
		CreatedAt:    metadata.CreatedAt,
		UpdatedAt:    metadata.UpdatedAt,
		ThumbnailURL: metadata.ThumbnailURL,
		VideoURL:     metadata.VideoURL,
	})
}
