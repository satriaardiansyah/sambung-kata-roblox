package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/satriaardiansyah/sambung-kata-roblox/db"
	"github.com/satriaardiansyah/sambung-kata-roblox/middleware"
	"github.com/satriaardiansyah/sambung-kata-roblox/shared"
)

// DashboardViewHandler renders the main dashboard page
func DashboardViewHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		http.ServeFile(w, r, "./templates/dashboard.html")
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// UserInfoAPIHandler returns the current authenticated user's profile
func UserInfoAPIHandler(w http.ResponseWriter, r *http.Request) {
	session, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var apiToken string
	err := db.DB.QueryRow("SELECT api_token FROM users WHERE id = ?", session.UserID).Scan(&apiToken)
	if err != nil {
		http.Error(w, `{"error": "Database error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"username":  session.Username,
		"role":      session.Role,
		"api_token": apiToken,
	})
}

// DashboardStatsAPIHandler calculates stats for dashboard cards
func DashboardStatsAPIHandler(w http.ResponseWriter, r *http.Request) {
	var totalUsers, killerSuffixesCount, deletedWordsCount, suggestedSuffixesCount int

	db.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&totalUsers)
	db.DB.QueryRow("SELECT COUNT(*) FROM killer_suffixes").Scan(&killerSuffixesCount)
	db.DB.QueryRow("SELECT COUNT(*) FROM deleted_words").Scan(&deletedWordsCount)
	db.DB.QueryRow("SELECT COUNT(*) FROM suggested_suffixes WHERE status = 'pending'").Scan(&suggestedSuffixesCount)

	activeSseClients := shared.GetConnectedClientsCount()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_users":              totalUsers,
		"active_sse_clients":       activeSseClients,
		"killer_suffixes_count":    killerSuffixesCount,
		"deleted_words_count":      deletedWordsCount,
		"suggested_suffixes_count": suggestedSuffixesCount,
	})
}

// KillerSuffixesAPIHandler handles GET (list), POST (add/update), and DELETE (remove) for suffixes
func KillerSuffixesAPIHandler(w http.ResponseWriter, r *http.Request) {
	session, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		suffixes, err := db.LoadKillerSuffixes()
		if err != nil {
			http.Error(w, `{"error": "Failed to load suffixes"}`, http.StatusInternalServerError)
			return
		}
		
		type SuffixItem struct {
			Suffix string `json:"suffix"`
			Score  int    `json:"score"`
		}
		
		var list []SuffixItem
		for k, v := range suffixes {
			list = append(list, SuffixItem{Suffix: k, Score: v})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(list)

	case http.MethodPost:
		if session.Role != "admin" {
			http.Error(w, `{"error": "Only admins can modify killer suffixes"}`, http.StatusForbidden)
			return
		}

		var req struct {
			Suffix string `json:"suffix"`
			Score  int    `json:"score"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error": "Invalid input"}`, http.StatusBadRequest)
			return
		}

		req.Suffix = strings.ToLower(strings.TrimSpace(req.Suffix))
		if req.Suffix == "" || req.Score <= 0 {
			http.Error(w, `{"error": "Suffix and positive score are required"}`, http.StatusBadRequest)
			return
		}

		err := db.AddOrUpdateKillerSuffix(req.Suffix, req.Score)
		if err != nil {
			http.Error(w, `{"error": "Failed to save suffix"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})

	case http.MethodDelete:
		if session.Role != "admin" {
			http.Error(w, `{"error": "Only admins can delete killer suffixes"}`, http.StatusForbidden)
			return
		}

		suffix := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("suffix")))
		if suffix == "" {
			http.Error(w, `{"error": "Suffix parameter is required"}`, http.StatusBadRequest)
			return
		}

		err := db.DeleteKillerSuffix(suffix)
		if err != nil {
			http.Error(w, `{"error": "Failed to delete suffix"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// DeletedWordsAPIHandler handles GET (list), POST (delete a new word), and DELETE (restore a word)
func DeletedWordsAPIHandler(w http.ResponseWriter, r *http.Request) {
	session, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		words, err := db.GetDeletedWords()
		if err != nil {
			http.Error(w, `{"error": "Failed to load deleted words"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(words)

	case http.MethodPost:
		var req struct {
			Word string `json:"word"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error": "Invalid input"}`, http.StatusBadRequest)
			return
		}

		req.Word = strings.ToLower(strings.TrimSpace(req.Word))
		if req.Word == "" {
			http.Error(w, `{"error": "Word is required"}`, http.StatusBadRequest)
			return
		}

		err := db.DeleteWord(req.Word)
		if err != nil {
			http.Error(w, `{"error": "Failed to blacklist word"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})

	case http.MethodDelete:
		if session.Role != "admin" {
			http.Error(w, `{"error": "Only admins can restore deleted words"}`, http.StatusForbidden)
			return
		}

		word := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("word")))
		if word == "" {
			http.Error(w, `{"error": "Word parameter is required"}`, http.StatusBadRequest)
			return
		}

		err := db.RestoreWord(word)
		if err != nil {
			http.Error(w, `{"error": "Failed to restore word"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// SuggestedSuffixesAPIHandler handles GET (list), POST (approve/reject), and DELETE (remove) suggested suffixes
func SuggestedSuffixesAPIHandler(w http.ResponseWriter, r *http.Request) {
	session, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		list, err := db.GetSuggestedSuffixes()
		if err != nil {
			http.Error(w, `{"error": "Failed to load suggested suffixes"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(list)

	case http.MethodPost:
		if session.Role != "admin" {
			http.Error(w, `{"error": "Only admins can approve suggestions"}`, http.StatusForbidden)
			return
		}

		var req struct {
			Query  string `json:"query"`
			Action string `json:"action"` // "approve" or "reject"
			Score  int    `json:"score"`  // optional score if approving
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error": "Invalid input"}`, http.StatusBadRequest)
			return
		}

		req.Query = strings.ToLower(strings.TrimSpace(req.Query))
		if req.Query == "" {
			http.Error(w, `{"error": "Query is required"}`, http.StatusBadRequest)
			return
		}

		if req.Action == "approve" {
			score := req.Score
			if score <= 0 {
				score = 500 // default score if not specified
			}
			// Add/Update in killer suffixes
			err := db.AddOrUpdateKillerSuffix(req.Query, score)
			if err != nil {
				http.Error(w, `{"error": "Failed to approve suffix"}`, http.StatusInternalServerError)
				return
			}
			// Update suggestion status
			db.UpdateSuggestedStatus(req.Query, "approved")
		} else if req.Action == "reject" {
			db.UpdateSuggestedStatus(req.Query, "rejected")
		} else {
			http.Error(w, `{"error": "Invalid action, must be 'approve' or 'reject'"}`, http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})

	case http.MethodDelete:
		if session.Role != "admin" {
			http.Error(w, `{"error": "Only admins can delete suggestions"}`, http.StatusForbidden)
			return
		}

		query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("query")))
		if query == "" {
			http.Error(w, `{"error": "Query parameter is required"}`, http.StatusBadRequest)
			return
		}

		err := db.DeleteSuggestedSuffix(query)
		if err != nil {
			http.Error(w, `{"error": "Failed to delete suggested suffix"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// TypingHistoryAPIHandler handles GET (list records) and POST (save typing test result)
func TypingHistoryAPIHandler(w http.ResponseWriter, r *http.Request) {
	session, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		history, err := db.GetTypingHistory(session.UserID)
		if err != nil {
			http.Error(w, `{"error": "Failed to load typing history"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(history)

	case http.MethodPost:
		var req struct {
			WPM      string `json:"wpm"`
			Accuracy string `json:"accuracy"`
			Duration string `json:"duration"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error": "Invalid input"}`, http.StatusBadRequest)
			return
		}

		wpmVal, err1 := strconv.ParseFloat(req.WPM, 64)
		accVal, err2 := strconv.ParseFloat(req.Accuracy, 64)
		durVal, err3 := strconv.Atoi(req.Duration)

		if err1 != nil || err2 != nil || err3 != nil {
			http.Error(w, `{"error": "Invalid typing values. WPM, Accuracy, and Duration must be numbers."}`, http.StatusBadRequest)
			return
		}

		err := db.SaveTypingRecord(session.UserID, wpmVal, accVal, durVal)
		if err != nil {
			http.Error(w, `{"error": "Failed to save typing record"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
