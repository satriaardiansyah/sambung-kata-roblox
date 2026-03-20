package main

import (
	// "bufio"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
)

var words []string

// func loadKamus() {
// 	file, err := os.Open("serang.txt")
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	defer file.Close()

// 	scanner := bufio.NewScanner(file)

// 	for scanner.Scan() {
// 		words = append(words, strings.ToLower(scanner.Text()))
// 	}
// }

var killerSuffix = map[string]int{
	"cy": 130,
	"gy": 90,
	"ex": 120,
	"rs": 70,
	"ks": 70,
	"ea": 60,
	"ly": 60,
	"tt": 80,
	"oo": 80,
	"mp": 90,
	"x" : 60,
	"ia": 100,
	"oi": 120,
	"pp": 100,
}

func loadKamus() {
	file, err := os.Open("kamus.json")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)

	err = decoder.Decode(&words)
	if err != nil {
		log.Fatal(err)
	}
}

var prefixIndex = map[string][]string{}

func buildIndex() {
	for _, w := range words {
		if len(w) >= 2 {
			key := w[:2]
			prefixIndex[key] = append(prefixIndex[key], w)
		}
	}
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	query := strings.ToLower(r.URL.Query().Get("q"))
	mode := r.URL.Query().Get("mode")

	type WordScore struct {
		Word  string
		Score int
	}

	var scored []WordScore

	for _, word := range words {

		match := false

		if mode == "prefix" && strings.HasPrefix(word, query) {
			match = true
		}

		if mode == "suffix" && strings.HasSuffix(word, query) {
			match = true
		}

		if !match {
			continue
		}

		if len(word) < 2 {
			continue
		}

		end := word[len(word)-2:]

		// base score (semakin kecil semakin bagus)
		score := 1000

		// 🔥 PRIORITAS KILLER SUFFIX
		if bonus, ok := killerSuffix[end]; ok {
			score -= bonus
		}

		scored = append(scored, WordScore{
			Word:  word,
			Score: score,
		})
	}

	// sorting
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score < scored[j].Score
	})

	var result []string
	for _, s := range scored {
		result = append(result, s.Word)
	}

	json.NewEncoder(w).Encode(result)
}

func getByPrefix(prefix string) []string {
	if val, ok := prefixIndex[prefix]; ok {
		return val
	}
	return []string{}
}

func bestMoveAdvanced(suffix string) []string {

	candidates := getByPrefix(suffix)

	type WordScore struct {
		Word  string
		Score int
	}

	var scored []WordScore

	for _, word := range candidates {

		if len(word) < 2 {
			continue
		}
		end := word[len(word)-2:]

		// kemungkinan lawan
		opponentMoves := getByPrefix(end)

		// kalau lawan tidak punya jawaban → AUTO WIN
		if len(opponentMoves) == 0 {
			return []string{word}
		}

		// hitung peluang kita setelah lawan jawab
		totalNext := 0

		for _, op := range opponentMoves {
			opEnd := op[len(op)-2:]
			next := getByPrefix(opEnd)
			totalNext += len(next)
		}

		score := len(opponentMoves)*100 - totalNext

		scored = append(scored, WordScore{
			Word:  word,
			Score: score,
		})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score < scored[j].Score
	})

	var result []string
	for i := 0; i < len(scored) && i < 10; i++ {
		result = append(result, scored[i].Word)
	}

	return result
}

func aiHandler(w http.ResponseWriter, r *http.Request) {

	query := strings.ToLower(r.URL.Query().Get("q"))

	if len(query) < 2 {
		json.NewEncoder(w).Encode([]string{})
		return
	}

	suffix := query[len(query)-2:]

	result := bestMoveAdvanced(suffix)

	json.NewEncoder(w).Encode(result)
}

func main() {

	loadKamus()
	buildIndex()

	http.Handle("/", http.FileServer(http.Dir("./templates")))

	http.HandleFunc("/search", searchHandler)
	http.HandleFunc("/ai", aiHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	log.Println("Server running on :" + port)

	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal(err)
	}
}