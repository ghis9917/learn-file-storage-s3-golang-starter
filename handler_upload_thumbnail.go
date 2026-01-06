package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

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

	const maxMemory = 10 << 20
	if err = r.ParseMultipartForm(maxMemory); err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't parse request", err)
		return
	}

	fileData, fileHeaderData, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't parse request", err)
		return
	}

	mediaType := fileHeaderData.Header.Get("Content-Type")

	thumbnailData, err := io.ReadAll(fileData)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't read file", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't find video", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "You're not the owner of the video", err)
		return
	}

	extension := strings.Split(mediaType, "/")
	randID := make([]byte, 32)
	rand.Read(randID)
	randIDString := base64.StdEncoding.EncodeToString(randID)

	savingDestination := filepath.Join(cfg.assetsRoot, fmt.Sprintf("%s.%s", randIDString, extension[len(extension)-1]))

	file, err := os.Create(savingDestination)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error while writing file", err)
		return
	}

	if _, err := file.Write(thumbnailData); err != nil {
		respondWithError(w, http.StatusInternalServerError, "error while writing file", err)
		return
	}

	thumbnailURL := fmt.Sprintf("http://localhost:%s/%s", cfg.port, savingDestination)

	newVideoMetaData := database.Video{
		ID:                video.ID,
		CreatedAt:         video.CreatedAt,
		UpdatedAt:         time.Now(),
		ThumbnailURL:      &thumbnailURL,
		VideoURL:          video.VideoURL,
		CreateVideoParams: video.CreateVideoParams,
	}
	if err = cfg.db.UpdateVideo(newVideoMetaData); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update file metadata", err)
		return
	}

	respondWithJSON(w, http.StatusOK, newVideoMetaData)
}
