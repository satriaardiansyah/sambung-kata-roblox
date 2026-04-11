package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
)

//go:embed kamus.json
var kamusData []byte

var words []string
var suffixIndex = map[string][]string{}

var killerSuffix = map[string]int{
	"cy": 130,
	"gy": 170,
	"ex": 120,
	"eo": 120,
	// "rs": 110,
	"ks": 180,
	// "ea": 50,
	// "ly": 140,
	// "tt": 140,
	"oo": 140,
	// "mp": 90,
	"x":  60,
	"ia": 60,
	"oi": 120,
	"pp": 100,
	"yab": 200,
	"iki": 200,
	"ipe": 200,
	"voi": 200,
	"coe": 200,
	"ez": 200,
	"ou": 200,
	"ox": 150,
	"tl": 200,
	"moi": 200,
	"sm": 210,
	"huh":250,
	"iya": 220,
	"dot": 250,
	"pei": 250,
	"ksa": 290,
	"ng": 60,
	"ml": 260,
	"sih": 260,
	"hih": 260,
	"meh": 300,
	"owa": 300,
	"esi": 260,
	// "uo": 200,

	// "ica": 140,
}

func loadKamus() {
	err := json.Unmarshal(kamusData, &words)
	if err != nil {
		log.Fatal("Gagal parse kamus.json: ", err)
	}
	fmt.Printf("Kamus berhasil dimuat: %d kata\n", len(words))
}

var prefixIndex = map[string][]string{}

func buildIndex() {
	for _, w := range words {
		if len(w) >= 2 {
			prefix := w[:2]
			suffix := w[len(w)-2:]

			prefixIndex[prefix] = append(prefixIndex[prefix], w)
			suffixIndex[suffix] = append(suffixIndex[suffix], w)
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

	var candidates []string

	//ambil kandidat dari index (BUKAN scan semua)
	if len(query) >= 2 {
		if mode == "prefix" {
			candidates = prefixIndex[query[:2]]
		} else if mode == "suffix" {
			candidates = suffixIndex[query[len(query)-2:]]
		}
	} else {
		// fallback kalau query pendek
		candidates = words
	}

	var scored []WordScore

	for _, word := range candidates {

		// filter sesuai mode
		if mode == "prefix" && !strings.HasPrefix(word, query) {
			continue
		}
		if mode == "suffix" && !strings.HasSuffix(word, query) {
			continue
		}

		if len(word) < 2 {
			continue
		}

		score := 1000

		// cek 3 huruf dulu (prioritas lebih spesifik)
		if len(word) >= 3 {
			end3 := word[len(word)-3:]
			if bonus, ok := killerSuffix[end3]; ok {
				score -= bonus
			}
		}

		// cek 2 huruf
		if len(word) >= 2 {
			end2 := word[len(word)-2:]
			if bonus, ok := killerSuffix[end2]; ok {
				score -= bonus
			}
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

	//limit hasil (biar hemat)
	limit := 50
	var result []string
	for i := 0; i < len(scored) && i < limit; i++ {
		result = append(result, scored[i].Word)
	}

	w.Header().Set("Content-Type", "application/json")
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
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]string{})
		return
	}

	suffix := query[len(query)-2:]
	result := bestMoveAdvanced(suffix)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func main() {
    // 1. Load data dulu
    loadKamus()
    buildIndex()

	// halaman utama
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "./templates/index2.html")
	})

	// halaman kedua
	http.HandleFunc("/page2", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./templates/index.html")
	})

	// API
	http.HandleFunc("/search", searchHandler)
	http.HandleFunc("/ai", aiHandler)

    // 3. Baru listen
    port := os.Getenv("PORT")
    if port == "" {
        port = "8000"
    }

    fmt.Println("PORT env value:", port)
    fmt.Println("Listening on 0.0.0.0:" + port)

    err := http.ListenAndServe("0.0.0.0:"+port, nil)
    if err != nil {
        log.Fatal(err)
    }
}