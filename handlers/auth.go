package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/satriaardiansyah/sambung-kata-roblox/db"
	"github.com/satriaardiansyah/sambung-kata-roblox/middleware"
	"golang.org/x/crypto/bcrypt"
)

func generateToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func LoginViewHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		http.ServeFile(w, r, "./templates/login.html")
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func LoginAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid JSON input"}`, http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, `{"error": "Username and password are required"}`, http.StatusBadRequest)
		return
	}

	var id int
	var passwordHash, role string
	err := db.DB.QueryRow("SELECT id, password_hash, role FROM users WHERE username = ?", req.Username).Scan(&id, &passwordHash, &role)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, `{"error": "Invalid username or password"}`, http.StatusUnauthorized)
		} else {
			http.Error(w, `{"error": "Database error"}`, http.StatusInternalServerError)
		}
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password))
	if err != nil {
		http.Error(w, `{"error": "Invalid username or password"}`, http.StatusUnauthorized)
		return
	}

	// Session token generation
	sessionToken := generateToken()
	expiresAt := time.Now().Add(24 * 30 * time.Hour) // 30 days session

	_, err = db.DB.Exec("INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)", sessionToken, id, expiresAt)
	if err != nil {
		http.Error(w, `{"error": "Failed to create session"}`, http.StatusInternalServerError)
		return
	}

	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		Expires:  expiresAt,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "success",
		"redirect": "/dashboard",
	})
}

func RegisterViewHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		http.ServeFile(w, r, "./templates/register.html")
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func RegisterAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid JSON input"}`, http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, `{"error": "Username and password are required"}`, http.StatusBadRequest)
		return
	}

	if len(req.Password) < 6 {
		http.Error(w, `{"error": "Password must be at least 6 characters"}`, http.StatusBadRequest)
		return
	}

	// Check if user already exists
	var dummy int
	err := db.DB.QueryRow("SELECT id FROM users WHERE username = ?", req.Username).Scan(&dummy)
	if err == nil {
		http.Error(w, `{"error": "Username already exists"}`, http.StatusBadRequest)
		return
	} else if err != sql.ErrNoRows {
		http.Error(w, `{"error": "Database error"}`, http.StatusInternalServerError)
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, `{"error": "Hashing error"}`, http.StatusInternalServerError)
		return
	}

	apiToken := generateToken()
	_, err = db.DB.Exec("INSERT INTO users (username, password_hash, role, api_token) VALUES (?, ?, 'user', ?)", req.Username, string(hashed), apiToken)
	if err != nil {
		http.Error(w, `{"error": "Failed to register user"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "success",
		"redirect": "/login",
	})
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		db.DB.Exec("DELETE FROM sessions WHERE token = ?", cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func RegenerateTokenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	newToken := generateToken()
	_, err := db.DB.Exec("UPDATE users SET api_token = ? WHERE id = ?", newToken, session.UserID)
	if err != nil {
		http.Error(w, `{"error": "Failed to update token"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":    "success",
		"api_token": newToken,
	})
}
