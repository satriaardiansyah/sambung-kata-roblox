package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/satriaardiansyah/sambung-kata-roblox/db"
)

type contextKey string

const (
	UserContextKey contextKey = "user"
)

type UserSession struct {
	UserID   int
	Username string
	Role     string
	Token    string
}

// SessionRequired middleware protects web-facing pages (dashboard, practice, etc.)
func SessionRequired(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		var session UserSession
		var expiresAt time.Time

		query := `
			SELECT u.id, u.username, u.role, s.token, s.expires_at 
			FROM sessions s
			JOIN users u ON s.user_id = u.id
			WHERE s.token = ?
		`
		err = db.DB.QueryRow(query, cookie.Value).Scan(
			&session.UserID, &session.Username, &session.Role, &session.Token, &expiresAt,
		)

		if err != nil {
			// Clear invalid cookie
			http.SetCookie(w, &http.Cookie{
				Name:     "session_token",
				Value:    "",
				Path:     "/",
				MaxAge:   -1,
				HttpOnly: true,
			})
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		if time.Now().After(expiresAt) {
			// Session expired, delete from database and cookie
			db.DB.Exec("DELETE FROM sessions WHERE token = ?", cookie.Value)
			http.SetCookie(w, &http.Cookie{
				Name:     "session_token",
				Value:    "",
				Path:     "/",
				MaxAge:   -1,
				HttpOnly: true,
			})
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Inject user session into context
		ctx := context.WithValue(r.Context(), UserContextKey, session)
		next(w, r.WithContext(ctx))
	}
}

// SessionOrAPIKeyRequired allows access if there is a valid web session (cookie) OR a valid API token
func SessionOrAPIKeyRequired(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Try checking for API Token first
		token := r.URL.Query().Get("token")
		if token == "" {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if token != "" {
			var userID int
			var username, role string
			err := db.DB.QueryRow("SELECT id, username, role FROM users WHERE api_token = ?", token).Scan(&userID, &username, &role)
			if err == nil {
				// Valid API Token
				ctx := context.WithValue(r.Context(), UserContextKey, UserSession{
					UserID:   userID,
					Username: username,
					Role:     role,
				})
				next(w, r.WithContext(ctx))
				return
			}
		}

		// 2. If no valid API Token, try checking for cookie Session
		cookie, err := r.Cookie("session_token")
		if err == nil {
			var session UserSession
			var expiresAt time.Time
			query := `
				SELECT u.id, u.username, u.role, s.token, s.expires_at 
				FROM sessions s
				JOIN users u ON s.user_id = u.id
				WHERE s.token = ?
			`
			err = db.DB.QueryRow(query, cookie.Value).Scan(
				&session.UserID, &session.Username, &session.Role, &session.Token, &expiresAt,
			)

			if err == nil && time.Now().Before(expiresAt) {
				// Valid session
				ctx := context.WithValue(r.Context(), UserContextKey, session)
				next(w, r.WithContext(ctx))
				return
			}
		}

		// 3. Unauthorized
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "Unauthorized: Valid web session or API token ?token=xxx is required."}`))
	}
}

// GetUserFromContext helper retrieves user info from the request context
func GetUserFromContext(r *http.Request) (UserSession, bool) {
	session, ok := r.Context().Value(UserContextKey).(UserSession)
	return session, ok
}
