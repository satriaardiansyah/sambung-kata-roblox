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

//go:embed kbbi_filtered.json
var kamusData []byte

var words []string
var suffixIndex = map[string][]string{}

var killerSuffix = map[string]int{
	"cy": 130,
	"gy": 170,
	"ex": 120,
	"eo": 120,
	"ks": 180,
	"oo": 140,
	"x":  60,
	"z":  60,
	"q":  60,
	"w": 60,
	"c": 60,
	"F": 60,
	"V": 60,
	"ia": 60,
	"oi": 120,
	"pp": 100,
	"iu": 200,
	"eh": 100,
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
	"meh": 260,
	"owa": 260,
	"esi": 260,
	"iat": 350,
	"ngeh": 270,
	"ngih": 270,
	"nggar": 300,
	"wati": 300,
	"inggu": 300,
	"logis": 400,
	"genik": 400,
	"alah": 300,
	"ngoh": 300,
	"tiol": 350,
	"taat": 400,
	"stis": 300,
	"kanya": 300,
	"angus": 300,
	"riksa": 400,
	"fault": 400,
	"burma": 400,
	// "uo": 200,

	// "ica": 140,
}

func loadKamus() {
	err := json.Unmarshal(kamusData, &words)
	if err != nil {
		log.Fatal("Gagal parse kbbi_consolidated.json: ", err)
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
	searchMode := r.URL.Query().Get("searchMode")

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

		// ✅ penalti panjang kata
		lengthDiff := len(word) - len(query)
		if lengthDiff > 0 {
			score += lengthDiff * 5
		}

		//cek 5 huruf

		if searchMode == "brutal" {
			if len(word) >= 5 {
				end4 := word[len(word)-5:]
				if bonus, ok := killerSuffix[end4]; ok {
					score -= bonus
				}
			}

			// cek 4 huruf
			if len(word) >= 4 {
				end4 := word[len(word)-4:]
				if bonus, ok := killerSuffix[end4]; ok {
					score -= bonus
				}
			}
		}

		

		// cek 3 huruf
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

		if len(word) >= 1 {
			end1 := word[len(word)-1:]
			if bonus, ok := killerSuffix[end1]; ok {
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

// =============================================
// DATA STRUCTURES (tambahkan di level package)
// =============================================

var prefixFrequency = map[string]int{}   // berapa kata berawalan X
var deadEndSuffixes = map[string]bool{}  // suffix yang tidak bisa disambung

func buildSmartIndex() {
	// Hitung frekuensi prefix dari kamus nyata
	for _, w := range words {
		for l := 1; l <= 4 && l <= len(w); l++ {
			prefixFrequency[w[:l]]++
		}
	}

	// Dead-end detection: suffix yang tidak ada satupun kata berawalan itu
	allSuffixes := map[string]bool{}
	for _, w := range words {
		for l := 1; l <= 4 && l <= len(w); l++ {
			suffix := w[len(w)-l:]
			allSuffixes[suffix] = false
		}
	}

	for suffix := range allSuffixes {
		freq := prefixFrequency[suffix]
		if freq == 0 {
			deadEndSuffixes[suffix] = true
		}
	}

	fmt.Printf("Dead-end suffixes ditemukan: %d\n", len(deadEndSuffixes))
}

// =============================================
// HELPER: konversi frekuensi prefix → bonus
// =============================================

func frequencyToBonus(freq int) int {
	switch {
	case freq == 0:
		return 600 // tidak ada kata yang bisa nyambung → jackpot
	case freq <= 3:
		return 450
	case freq <= 10:
		return 350
	case freq <= 30:
		return 250
	case freq <= 100:
		return 150
	case freq <= 300:
		return 80
	case freq <= 600:
		return 40
	default:
		return 0
	}
}

// =============================================
// SEARCH HANDLER BARU
// =============================================

func searchHandlerV2(w http.ResponseWriter, r *http.Request) {
	query := strings.ToLower(r.URL.Query().Get("q"))
	mode := r.URL.Query().Get("mode")
	searchMode := r.URL.Query().Get("searchMode")

	type WordScore struct {
		Word  string
		Score int
	}

	// --- Ambil kandidat dari index ---
	var candidates []string
	if len(query) >= 2 {
		if mode == "prefix" {
			candidates = prefixIndex[query[:2]]
		} else if mode == "suffix" {
			candidates = suffixIndex[query[len(query)-2:]]
		}
	} else if len(query) == 1 {
		// Query 1 huruf: scan prefixIndex yang berawalan huruf itu
		for key, list := range prefixIndex {
			if strings.HasPrefix(key, query) {
				candidates = append(candidates, list...)
			}
		}
	} else {
		candidates = words
	}

	var scored []WordScore

	for _, word := range candidates {
		// Filter ketat sesuai mode
		if mode == "prefix" && !strings.HasPrefix(word, query) {
			continue
		}
		if mode == "suffix" && !strings.HasSuffix(word, query) {
			continue
		}
		if len(word) < 3 {
			continue
		}

		score := 1000

		// --- Penalti panjang kata (lebih panjang = lebih susah dilawan) ---
		lengthDiff := len(word) - len(query)
		if lengthDiff > 0 {
			score += lengthDiff * 3
		}

		// --- LAYER 1: Dead-end detection (paling powerful) ---
		// Kalau suffix kata ini tidak ada satupun kata lain yang bisa nyambung → bonus besar
		for suffixLen := 1; suffixLen <= 4 && suffixLen <= len(word); suffixLen++ {
			suffix := word[len(word)-suffixLen:]
			if deadEndSuffixes[suffix] {
				// Makin pendek dead-end suffix, makin susah dilawan
				switch suffixLen {
				case 1:
					score -= 700 // 1 huruf akhir buntu → sangat mematikan
				case 2:
					score -= 400
				case 3:
					score -= 200
				case 4:
					score -= 100
				}
				break // ambil dead-end terpendek saja, sudah cukup
			}
		}

		// --- LAYER 2: Scoring berbasis frekuensi prefix nyata (akumulatif) ---
		// Cek semua layer suffix 1-4 huruf, akumulasi bonusnya
		for suffixLen := 1; suffixLen <= 4 && suffixLen <= len(word); suffixLen++ {
			suffix := word[len(word)-suffixLen:]
			freq := prefixFrequency[suffix]
			bonus := frequencyToBonus(freq)

			// Suffix pendek lebih berpengaruh (huruf akhir langsung jadi awalan lawan)
			weight := 1.0
			switch suffixLen {
			case 1:
				weight = 1.5
			case 2:
				weight = 1.2
			case 3:
				weight = 0.8
			case 4:
				weight = 0.5
			}

			score -= int(float64(bonus) * weight)
		}

		// --- LAYER 3: killerSuffix manual (tetap dipakai sebagai boost tambahan) ---
		if searchMode == "brutal" {
			for suffixLen := 5; suffixLen >= 1; suffixLen-- {
				if suffixLen > len(word) {
					continue
				}
				end := word[len(word)-suffixLen:]
				if bonus, ok := killerSuffix[end]; ok {
					score -= bonus
					break // hanya ambil match terpanjang
				}
			}
		} else {
			// mode normal: cek 1-3 huruf saja
			for suffixLen := 3; suffixLen >= 1; suffixLen-- {
				if suffixLen > len(word) {
					continue
				}
				end := word[len(word)-suffixLen:]
				if bonus, ok := killerSuffix[end]; ok {
					score -= bonus
					break
				}
			}
		}

		scored = append(scored, WordScore{Word: word, Score: score})
	}

	// --- Sort: score terkecil = paling mematikan ---
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].Score == scored[j].Score {
			// Tie-break: kata lebih panjang lebih diutamakan
			return len(scored[i].Word) > len(scored[j].Word)
		}
		return scored[i].Score < scored[j].Score
	})

	// Limit hasil
	limit := 50
	var result []string
	for i := 0; i < len(scored) && i < limit; i++ {
		result = append(result, scored[i].Word)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func main() {
    // 1. Load data dulu
    loadKamus()
    buildIndex()
	buildSmartIndex()

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
	http.HandleFunc("/search2",  searchHandlerV2)

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