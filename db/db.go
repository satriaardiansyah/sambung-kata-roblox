package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"crypto/rand"
	"encoding/hex"

	_ "modernc.org/sqlite"
	"golang.org/x/crypto/bcrypt"
)

var DB *sql.DB

// Thread-safe memory cache for fast search queries
var (
	cacheMu            sync.RWMutex
	KillerSuffixCache  = map[string]int{}
	DeletedWordsCache  = map[string]bool{}
)

// InitDB initializes the SQLite database, runs migrations, and seeds default data.
func InitDB(dbPath string) {
	if dbPath == "" {
		dbPath = os.Getenv("DB_PATH")
		if dbPath == "" {
			dbPath = "./data.db"
		}
	}

	// Ensure parent directory exists
	dir := filepath.Dir(dbPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Failed to create database directory: %v", err)
		}
	}

	var err error
	DB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("Failed to open SQLite database: %v", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := DB.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		log.Printf("Warning: Failed to set WAL mode: %v", err)
	}

	migrate()
	seed()
	
	// Initial cache load
	if err := ReloadCache(); err != nil {
		log.Fatalf("Failed to populate in-memory cache: %v", err)
	}
}

func migrate() {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'user',
			api_token TEXT UNIQUE NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS sessions (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			expires_at DATETIME NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS killer_suffixes (
			suffix TEXT PRIMARY KEY,
			score INTEGER NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS deleted_words (
			word TEXT PRIMARY KEY,
			deleted_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS suggested_suffixes (
			query TEXT PRIMARY KEY,
			prefix_count INTEGER NOT NULL,
			prefix_words TEXT NOT NULL, -- JSON string or comma-separated list
			suffix_count INTEGER NOT NULL,
			suffix_words TEXT NOT NULL, -- JSON string or comma-separated list
			hits INTEGER DEFAULT 1,
			status TEXT DEFAULT 'pending'
		);`,
		`CREATE TABLE IF NOT EXISTS typing_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			wpm REAL NOT NULL,
			accuracy REAL NOT NULL,
			duration_seconds INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);`,
	}

	for _, query := range queries {
		if _, err := DB.Exec(query); err != nil {
			log.Fatalf("Database migration failed for query:\n%s\nError: %v", query, err)
		}
	}
	fmt.Println("Database migrations applied successfully.")
}

func generateRandomToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "admin_default_token_12345"
	}
	return hex.EncodeToString(b)
}

func seed() {
	// 1. Seed default admin if no users exist
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		log.Fatalf("Failed to check user count: %v", err)
	}

	if count == 0 {
		hashed, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		if err != nil {
			log.Fatalf("Failed to hash default admin password: %v", err)
		}
		token := generateRandomToken()
		_, err = DB.Exec(
			"INSERT INTO users (username, password_hash, role, api_token) VALUES (?, ?, ?, ?)",
			"admin", string(hashed), "admin", token,
		)
		if err != nil {
			log.Fatalf("Failed to seed default admin: %v", err)
		}
		fmt.Printf("Seeded default admin user.\nUsername: admin\nPassword: admin123\nAPI Token: %s\n", token)
	}
}

// ReloadCache repopulates the in-memory cache from the SQLite database
func ReloadCache() error {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	// 1. Load killer suffixes
	rows, err := DB.Query("SELECT suffix, score FROM killer_suffixes")
	if err != nil {
		return err
	}
	defer rows.Close()

	newKillerSuffixes := make(map[string]int)
	for rows.Next() {
		var suffix string
		var score int
		if err := rows.Scan(&suffix, &score); err != nil {
			return err
		}
		newKillerSuffixes[suffix] = score
	}
	KillerSuffixCache = newKillerSuffixes

	// 2. Load deleted words
	rows2, err := DB.Query("SELECT word FROM deleted_words")
	if err != nil {
		return err
	}
	defer rows2.Close()

	newDeletedWords := make(map[string]bool)
	for rows2.Next() {
		var word string
		if err := rows2.Scan(&word); err != nil {
			return err
		}
		newDeletedWords[word] = true
	}
	DeletedWordsCache = newDeletedWords

	return nil
}

// Cache reader helper functions

func GetKillerSuffixesFromCache() map[string]int {
	cacheMu.RLock()
	defer cacheMu.RUnlock()

	// Return a copy to avoid concurrency/reference issues
	m := make(map[string]int, len(KillerSuffixCache))
	for k, v := range KillerSuffixCache {
		m[k] = v
	}
	return m
}

func IsWordDeletedInCache(word string) bool {
	cacheMu.RLock()
	defer cacheMu.RUnlock()
	return DeletedWordsCache[word]
}

// SeedInitialKillerSuffixes is called from main.go to load default suffixes if the DB is empty
func SeedInitialKillerSuffixes(defaultSuffixes map[string]int) {
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM killer_suffixes").Scan(&count)
	if err != nil {
		log.Printf("Failed to count killer suffixes: %v", err)
		return
	}

	if count > 0 {
		return
	}

	tx, err := DB.Begin()
	if err != nil {
		log.Printf("Failed to start transaction for seeding suffixes: %v", err)
		return
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO killer_suffixes (suffix, score) VALUES (?, ?)")
	if err != nil {
		log.Printf("Failed to prepare statement for seeding suffixes: %v", err)
		return
	}
	defer stmt.Close()

	for suffix, score := range defaultSuffixes {
		_, err := stmt.Exec(suffix, score)
		if err != nil {
			log.Printf("Failed to insert default suffix %s: %v", suffix, err)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction for seeding suffixes: %v", err)
	} else {
		fmt.Printf("Seeded %d default killer suffixes into SQLite.\n", len(defaultSuffixes))
		// Refresh cache
		ReloadCache()
	}
}

// Helper query utilities for global application use

func GetDeletedWords() ([]string, error) {
	rows, err := DB.Query("SELECT word FROM deleted_words")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []string
	for rows.Next() {
		var w string
		if err := rows.Scan(&w); err != nil {
			return nil, err
		}
		list = append(list, w)
	}
	return list, nil
}

func DeleteWord(word string) error {
	_, err := DB.Exec("INSERT OR IGNORE INTO deleted_words (word) VALUES (?)", word)
	if err == nil {
		ReloadCache()
	}
	return err
}

func RestoreWord(word string) error {
	_, err := DB.Exec("DELETE FROM deleted_words WHERE word = ?", word)
	if err == nil {
		ReloadCache()
	}
	return err
}

func LoadKillerSuffixes() (map[string]int, error) {
	rows, err := DB.Query("SELECT suffix, score FROM killer_suffixes")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := make(map[string]int)
	for rows.Next() {
		var suffix string
		var score int
		if err := rows.Scan(&suffix, &score); err != nil {
			return nil, err
		}
		m[suffix] = score
	}
	return m, nil
}

func AddOrUpdateKillerSuffix(suffix string, score int) error {
	_, err := DB.Exec("INSERT INTO killer_suffixes (suffix, score) VALUES (?, ?) ON CONFLICT(suffix) DO UPDATE SET score=excluded.score", suffix, score)
	if err == nil {
		ReloadCache()
	}
	return err
}

func DeleteKillerSuffix(suffix string) error {
	_, err := DB.Exec("DELETE FROM killer_suffixes WHERE suffix = ?", suffix)
	if err == nil {
		ReloadCache()
	}
	return err
}

func GetSuggestedSuffixes() ([]map[string]interface{}, error) {
	rows, err := DB.Query("SELECT query, prefix_count, prefix_words, suffix_count, suffix_words, hits, status FROM suggested_suffixes ORDER BY hits DESC, suffix_count DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []map[string]interface{}
	for rows.Next() {
		var query string
		var prefixCount, suffixCount, hits int
		var prefixWords, suffixWords, status string
		if err := rows.Scan(&query, &prefixCount, &prefixWords, &suffixCount, &suffixWords, &hits, &status); err != nil {
			return nil, err
		}
		
		result = append(result, map[string]interface{}{
			"query":        query,
			"prefix_count": prefixCount,
			"prefix_words":  strings.Split(prefixWords, ","),
			"suffix_count": suffixCount,
			"suffix_words":  strings.Split(suffixWords, ","),
			"hits":         hits,
			"status":       status,
		})
	}
	return result, nil
}

func SaveSuggestedSuffix(query string, prefixCount int, prefixWords []string, suffixCount int, suffixWords []string) error {
	prefStr := strings.Join(prefixWords, ",")
	suffStr := strings.Join(suffixWords, ",")

	_, err := DB.Exec(`
		INSERT INTO suggested_suffixes (query, prefix_count, prefix_words, suffix_count, suffix_words, hits, status)
		VALUES (?, ?, ?, ?, ?, 1, 'pending')
		ON CONFLICT(query) DO UPDATE SET hits = hits + 1
	`, query, prefixCount, prefStr, suffixCount, suffStr)
	return err
}

func UpdateSuggestedStatus(query string, status string) error {
	_, err := DB.Exec("UPDATE suggested_suffixes SET status = ? WHERE query = ?", status, query)
	return err
}

func DeleteSuggestedSuffix(query string) error {
	_, err := DB.Exec("DELETE FROM suggested_suffixes WHERE query = ?", query)
	return err
}

func SaveTypingRecord(userID int, wpm float64, accuracy float64, duration int) error {
	_, err := DB.Exec("INSERT INTO typing_history (user_id, wpm, accuracy, duration_seconds) VALUES (?, ?, ?, ?)", userID, wpm, accuracy, duration)
	return err
}

func GetTypingHistory(userID int) ([]map[string]interface{}, error) {
	rows, err := DB.Query("SELECT wpm, accuracy, duration_seconds, created_at FROM typing_history WHERE user_id = ? ORDER BY created_at DESC LIMIT 50", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []map[string]interface{}
	for rows.Next() {
		var wpm, accuracy float64
		var duration int
		var createdAt string // changed to string to handle SQLite date parsing easily
		if err := rows.Scan(&wpm, &accuracy, &duration, &createdAt); err != nil {
			return nil, err
		}
		
		// Parse string to custom formatted time or keep as string
		history = append(history, map[string]interface{}{
			"wpm":              wpm,
			"accuracy":         accuracy,
			"duration_seconds": duration,
			"created_at":       createdAt,
		})
	}
	return history, nil
}
