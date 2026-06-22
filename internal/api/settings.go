// This file implements GET/PUT /api/settings, currently just the
// Telegram bot's capture mode (plan_amethyst-telegram-bot §4).
package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/Stinger911/Amethyst/internal/index"
	"github.com/Stinger911/Amethyst/internal/settings"
)

// SettingsResponse is the JSON body of both GET and PUT /api/settings.
type SettingsResponse struct {
	CaptureMode string `json:"captureMode"`
}

// GetSettingsHandler serves GET /api/settings.
func GetSettingsHandler(db *index.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mode, err := settings.GetCaptureMode(db)
		if err != nil {
			log.Printf("get settings: %v", err)
			http.Error(w, "get settings failed", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, SettingsResponse{CaptureMode: mode})
	}
}

// SaveSettingsHandler serves PUT /api/settings.
func SaveSettingsHandler(db *index.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req SettingsResponse
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if err := settings.SetCaptureMode(db, req.CaptureMode); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, req)
	}
}
