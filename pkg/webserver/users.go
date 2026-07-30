// file: pkg/webserver/users.go
// version: 1.1.0
// guid: 3f14cff3-19b6-461e-9c49-592a737c1a7d
// last-edited: 2026-07-30

package webserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/spf13/viper"

	auth "github.com/jdfalk/subtitle-manager/pkg/gcommonauth"
	"github.com/jdfalk/subtitle-manager/pkg/notifications"
)

// usersHandler returns a list of all users.
func usersHandler(db *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		users, err := auth.ListUsers(db)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(users)
	})
}

// userResetHandler resets the password for the specified user ID, generates a new
// API key and emails the credentials using the configured notification service.
func userResetHandler(db *sql.DB) http.Handler {
	type response struct {
		Password string `json:"password"`
		APIKey   string `json:"api_key"`
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		parts := pathSegments(r, "/api/users/")
		if len(parts) != 2 || parts[1] != "reset" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		id, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil || id == 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		pass, key, err := auth.ResetPassword(db, id)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var email, username string
		row := db.QueryRow(`SELECT email, username FROM users WHERE id = ?`, id)
		_ = row.Scan(&email, &username)
		if email != "" {
			svc, err := notifications.New(
				viper.GetString("notifications.discord_webhook"),
				viper.GetString("notifications.telegram_token"),
				viper.GetString("notifications.telegram_chat_id"),
				viper.GetString("notifications.email_url"),
			)
			if err != nil {
				http.Error(w, fmt.Sprintf("Invalid notification configuration: %v", err), http.StatusInternalServerError)
				return
			}
			msg := fmt.Sprintf("Credentials for %s\nPassword: %s\nAPI Key: %s", username, pass, key)
			_ = svc.Send(context.Background(), msg)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response{Password: pass, APIKey: key})
	})
}

// userRouter dispatches user-related sub paths like password resets or tag management.
func userRouter(db *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		base := strings.Trim(viper.GetString("base_url"), "/")
		if base != "" {
			base = "/" + base
		}
		p := strings.TrimPrefix(r.URL.Path, base+"/api/users/")
		switch {
		case strings.HasSuffix(p, "/reset"):
			userResetHandler(db).ServeHTTP(w, r)
		case strings.Contains(p, "/tags"):
			userTagsHandler(db).ServeHTTP(w, r)
		case p == "" || p == "/":
			usersHandler(db).ServeHTTP(w, r)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
}
