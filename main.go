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
	"sync"
)

//go:embed kbbi_updated.json
var kamusData []byte

var words []string
var suffixIndex = map[string][]string{}

var deletedWords []string
var deletedMu    sync.Mutex
const deletedFile = "deleted_words.json"

// =============================================
// SUGGESTED SUFFIX LOGGER
// Menyimpan query yang berpotensi jadi "killerSuffix" baru:
// - panjang query > 3
// - sebagai PREFIX hasilnya cuma 1-3 kata (susah disambung lawan)
// - sebagai SUFFIX hasilnya >= 3 kata (banyak korban yang bisa kena)
// =============================================
var suggestedSuffixes = map[string]SuggestedSuffixEntry{}
var suggestedMu sync.Mutex
const suggestedFile = "suggested_suffixes.json"

type SuggestedSuffixEntry struct {
	Query        string   `json:"query"`
	PrefixCount  int      `json:"prefix_count"`
	PrefixWords  []string `json:"prefix_words"`
	SuffixCount  int      `json:"suffix_count"`
	SuffixWords  []string `json:"suffix_words"`
	Hits         int      `json:"hits"` // berapa kali query ini muncul lagi
}

func loadSuggestedSuffixes() {
	data, err := os.ReadFile(suggestedFile)
	if err != nil {
		return
	}
	json.Unmarshal(data, &suggestedSuffixes)
	fmt.Printf("Suggested suffixes dimuat: %d entri\n", len(suggestedSuffixes))
}

func saveSuggestedSuffixes() {
	data, _ := json.MarshalIndent(suggestedSuffixes, "", "  ")
	os.WriteFile(suggestedFile, data, 0644)
}

// maybeLogSuggestedSuffix mengecek kriteria dan menyimpan kalau cocok.
// Dipanggil dari searchHandler, pakai data words yang sudah ada (tanpa query ulang berat).
func maybeLogSuggestedSuffix(query string) {
	if len(query) <= 3 {
		return
	}

	// Hitung match sebagai prefix (awalan query)
	var prefixWords []string
	for _, w := range words {
		if strings.HasPrefix(w, query) {
			prefixWords = append(prefixWords, w)
		}
	}

	// Kriteria 1: awalan sedikit (1-3 kata)
	if len(prefixWords) < 1 || len(prefixWords) > 5 {
		return
	}

	// Hitung match sebagai suffix (akhiran query)
	var suffixWords []string
	for _, w := range words {
		if strings.HasSuffix(w, query) {
			suffixWords = append(suffixWords, w)
		}
	}

	// Kriteria 2: akhiran banyak (>= 3 kata)
	if len(suffixWords) < 2 {
		return
	}

	suggestedMu.Lock()
	defer suggestedMu.Unlock()

	if existing, ok := suggestedSuffixes[query]; ok {
		existing.Hits++
		suggestedSuffixes[query] = existing
	} else {
		suggestedSuffixes[query] = SuggestedSuffixEntry{
			Query:       query,
			PrefixCount: len(prefixWords),
			PrefixWords: prefixWords,
			SuffixCount: len(suffixWords),
			SuffixWords: suffixWords,
			Hits:        1,
		}
	}
	saveSuggestedSuffixes()
}

func suggestedSuffixHandler(w http.ResponseWriter, r *http.Request) {
	suggestedMu.Lock()
	defer suggestedMu.Unlock()

	items := make([]SuggestedSuffixEntry, 0, len(suggestedSuffixes))
	for _, v := range suggestedSuffixes {
		items = append(items, v)
	}

	// Urutkan: paling sering muncul (hits) di atas, lalu suffix_count terbanyak
	sort.Slice(items, func(i, j int) bool {
		if items[i].Hits != items[j].Hits {
			return items[i].Hits > items[j].Hits
		}
		return items[i].SuffixCount > items[j].SuffixCount
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func deleteSuggestedSuffixHandler(w http.ResponseWriter, r *http.Request) {
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	if query == "" {
		http.Error(w, "query kosong", 400)
		return
	}

	suggestedMu.Lock()
	defer suggestedMu.Unlock()

	if _, ok := suggestedSuffixes[query]; !ok {
		http.Error(w, "entri tidak ditemukan", 404)
		return
	}
	delete(suggestedSuffixes, query)
	saveSuggestedSuffixes()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "deleted", "query": query})
}

var killerSuffix = map[string]int{
	// 5 Karakter
	"orasi": 1000,
	"kisme": 1000,
	"sitas": 950,
	"takan": 950,
	"iskan": 950,
	"likan": 950,
	"litik": 950,
	"aksis": 900,
	"angsa": 1000,
	"ering": 400,
	"rodok": 1000,
	"inggi": 900,
	"abaka": 999,
	"stele": 999,
	"alari": 950,
	"anasi": 950,
	"entil": 950,
	"anser": 900,
	"nggor": 900,
	"tipus": 900,
	"ancar": 800,
	"andur": 800,
	"angus": 800,
	"ansor": 800,
	"antem": 800,
	"arong": 800,
	"arkil": 800,
	"awang": 800,
	"elli":  800,
	"ensil": 800,
	"ergot": 800,
	"hapak": 800,
	"kimah": 800,
	"lahad": 800,
	"latah": 800,
	"matis": 900,
	"ofoni": 800,
	"oleac": 800,
	"olong": 800,
	"ritis": 800,
	"tanai": 800,
	"tisis": 800,
	"tonik": 800,
	"ungsi": 800,
	"ahang": 580,
	"fauna": 580,
	"garot": 580,
	"gatot": 580,
	"mboli": 580,
	"ngudo": 580,
	"olang": 580,
	"sosro": 580,
	"amang": 550,
	"hohon": 550,
	"isian": 550,
	"riksa": 550,
	"trium": 550,
	"hiran": 500,
	"ganas": 450,
	"garpu": 450,
	"jijik": 450,
	"kolam": 450,
	"manat": 450,
	"meula": 450,
	"nusuk": 450,
	"ratif": 450,
	"umang": 450,
	"burma": 400,
	"fault": 400,
	"ruang": 400,
	"tikam": 400,
	"duksi": 300,
	"genik": 300,
	"kanya": 300,
	"logis": 300,
	"nggar": 300,
	"arian": 270,
	"meter": 100,
	"ogram": 100,

	// 4 Karakter
	"tusa": 800,
	"eran": 800,
	"atik": 800,
	"rian": 800,
	"iran": 550,
	"alah": 800,
	"unan": 999,
	"inkan": 400,
	"taat":  400,
	"tiol":  350,
	"ilok":  320,
	"lipe":  320,
	"anki":  300,
	"atat":  300,
	"epik":  300,
	"inggu": 300,
	"ngeh":  300,
	"ngoh":  300,
	"riko":  300,
	"stis":  300,
	"wati":  300,
	"anah":  250,
	"asel":  250,
	"apet":  200,

	// 3 Karakter
	"eni": 300,
	"ksa": 290,
	"esi": 260,
	"hih": 260,
	"meh": 260,
	"owa": 260,
	"sih": 260,
	"bou": 250,
	"dot": 250,
	"huh": 250,
	"pei": 250,
	"pso": 250,
	"iya": 220,
	"coe": 200,
	"iki": 200,
	"ipe": 200,
	"moi": 200,
	"voi": 200,
	"yab": 200,

	// 2 Karakter
	"ns": 300,
	"ml": 260,
	"sm": 210,
	"ez": 200,
	"iu": 200,
	"ou": 200,
	"tl": 200,
	"ks": 180,
	"gy": 170,
	"ox": 150,
	"oo": 140,
	"cy": 130,
	"eo": 120,
	"eq": 120,
	"ex": 120,
	"oi": 120,
	"eh": 100,
	"oh": 100,
	"pp": 100,
	"ts": 100,
	"ia": 60,
	"ng": 60,

	// 1 Karakter
	"F": 60,
	"V": 60,
	"c": 60,
	"q": 60,
	"w": 60,
	"x": 60,
	"z": 60,
}


// var killerSuffix = map[string]int{
// 	// 5 Karakter
// 	"aksis": 900,
// 	"angsa": 1000,
// 	"sitas": 1000,
// 	"rodok": 1000,
// 	"ering": 900,
// 	"abaka": 999,
// 	"stele": 999,
// 	"alari": 950,
// 	"anasi": 950,
// 	"entil": 950,
// 	"takan": 950,
// 	"iskan": 940,
// 	"likan": 930,
// 	"litik": 920,
// 	"erang": 910,
// 	"anser": 900,
// 	"awang": 900,
// 	"lahan": 900,
// 	"nggor": 900,
// 	"tipus": 900,
// 	"ratif": 890,
// 	"olong": 860,
// 	"matis": 850,
// 	"ritis": 840,
// 	"adang": 815,
// 	"ilang": 815,
// 	"iring": 815,
// 	"arung": 810,
// 	"etika": 810,
// 	"siasi": 810,
// 	"ulung": 805,
// 	"atang": 804,
// 	"tesis": 801,
// 	"inggi": 460,
// 	"inkan": 400,
// 	"inggu": 300,

// 	// 4 Karakter
// 	"eran": 1000,
// 	"alah": 980,
// 	"rian": 990,
// 	"unan": 970,
// 	"nggi": 880,
// 	"ural": 830,
// 	"anik": 820,
// 	"urat": 810,
// 	"ofon": 804,
// 	"tusa": 800,
// 	"iran": 550,
// 	"taat": 400,
// 	"tiol": 350,
// 	"ilok": 320,
// 	"lipe": 320,
// 	"anki": 300,
// 	"atat": 300,
// 	"epik": 300,
// 	"ngeh": 300,
// 	"ngoh": 300,
// 	"riko": 300,
// 	"stis": 300,
// 	"wati": 300,
// 	"anah": 250,
// 	"asel": 250,
// 	"apet": 200,

// 	// 3 Karakter
// 	"200": 350,
// 	"eni": 300,
// 	"ksa": 290,
// 	"esi": 260,
// 	"hih": 260,
// 	"meh": 260,
// 	"owa": 260,
// 	"sih": 260,
// 	"bou": 250,
// 	"dot": 250,
// 	"huh": 250,
// 	"pei": 250,
// 	"pso": 250,
// 	"iya": 220,
// 	"coe": 200,
// 	"iki": 200,
// 	"ipe": 200,
// 	"moi": 200,
// 	"voi": 200,
// 	"yab": 200,

// 	// 2 Karakter
// 	"ns": 300,
// 	"ml": 260,
// 	"sm": 210,
// 	"ez": 200,
// 	"iu": 200,
// 	"ou": 200,
// 	"tl": 200,
// 	"ks": 180,
// 	"gy": 170,
// 	"ox": 150,
// 	"oo": 140,
// 	"cy": 130,
// 	"eo": 120,
// 	"eq": 120,
// 	"ex": 120,
// 	"oi": 120,
// 	"eh": 100,
// 	"oh": 100,
// 	"pp": 100,
// 	"ts": 100,
// 	"ia": 60,
// 	"ng": 60,

// 	// 1 Karakter
// 	"F": 60,
// 	"V": 60,
// 	"c": 60,
// 	"q": 60,
// 	"w": 60,
// 	"x": 60,
// 	"z": 60,
// }




var killerOpener = map[string]int{
	"bouea": 0,
	"ofonik": 0,
	"aksismus": 0,
	"ansori": 0,
    "iranika":   0,
    "iranga":    0,
    "garpuan":   0,
	"olanggara": 0,
	"ahangkara": 0,
	"umangkapala": 0,
	"gatotkaca": 0,
	"tikaman": 0,
	"faunasia": 0,
	"faunal": 0,
	"tisisme": 0,
	"angsang": 0,
	 "tonikum,": 0,
	 "ikadabuki": 0,
	 "alarima": 0,
	 "arongan": 0,
	 "tipuse": 0,
	 "litikafobia": 0,
	 "riksaan": 0,
	 "nggore": 0,
	 "ratifikasi": 0,
	 "ergoterapi": 0,
	 "ergot": 0,
	 "dehal": 0,
}

var warningWords = map[string]int{
	"nggar": 0,
	"genik": 0,
	"logis": 0,
	"iran": 0,
	"iat": 0,
	"nggil": 0,
	"riksaan": 0,
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
	query      := strings.ToLower(r.URL.Query().Get("q"))
	mode       := r.URL.Query().Get("mode")
	searchMode := r.URL.Query().Get("searchMode")
	prioritas  := r.URL.Query().Get("prioritas") // ← TAMBAH INI

    // Parse prioritas jadi slice
    var prioritasList []string
    if prioritas != "" {
        for _, p := range strings.Split(prioritas, ",") {
            p = strings.TrimSpace(p)
            if p != "" {
                prioritasList = append(prioritasList, p)
            }
        }
    }

	type WordScore struct {
		Word  string
		Score int
	}

	var candidates []string
	if len(query) >= 2 {
		if mode == "prefix" {
			candidates = prefixIndex[query[:2]]
		} else if mode == "suffix" {
			candidates = suffixIndex[query[len(query)-2:]]
		}
	} else {
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

		if mode == "prefix" {
			if _, isOpener := killerOpener[word]; isOpener {
				score = -9999
			}
		}

		lengthDiff := len(word) - len(query)
		if lengthDiff > 0 {
			score += lengthDiff * 5
		}

		if searchMode == "brutal" {
			if len(word) >= 5 {
				if bonus, ok := killerSuffix[word[len(word)-5:]]; ok {
					score -= bonus
				}
			}
			if len(word) >= 4 {
				if bonus, ok := killerSuffix[word[len(word)-4:]]; ok {
					score -= bonus
				}
			}
		}
		if len(word) >= 3 {
			if bonus, ok := killerSuffix[word[len(word)-3:]]; ok {
				score -= bonus
			}
		}
		if len(word) >= 2 {
			if bonus, ok := killerSuffix[word[len(word)-2:]]; ok {
				score -= bonus
			}
		}
		if len(word) >= 1 {
			if bonus, ok := killerSuffix[word[len(word)-1:]]; ok {
				score -= bonus
			}
		}

		for _, pSuffix := range prioritasList {
            if strings.HasSuffix(word, pSuffix) {
                score -= 2000 // nilai sangat rendah = muncul paling atas
                break
            }
        }

		scored = append(scored, WordScore{Word: word, Score: score})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score < scored[j].Score
	})

	// Return []string biasa — warning logic ada di FE
	limit  := 50
	result := make([]string, 0, limit)
	for i := 0; i < len(scored) && i < limit; i++ {
		result = append(result, scored[i].Word)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)

	// Cek & simpan kandidat suffix baru (non-blocking biar ga ganggu response)
	go maybeLogSuggestedSuffix(query)
}

func dangerWordsHandler(w http.ResponseWriter, r *http.Request) {
    // Terima suffixes dari FE
    suffixes := strings.Split(r.URL.Query().Get("suffixes"), ",")
    
    result := map[string][]string{} // suffix → []kata
    
    for _, word := range words {
        wl := strings.ToLower(word)
        for _, suffix := range suffixes {
            suffix = strings.TrimSpace(suffix)
            if suffix == "" { continue }
            if strings.HasSuffix(wl, suffix) {
                result[suffix] = append(result[suffix], word)
            }
        }
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

// Channel untuk broadcast ke semua client
var sseClients = map[chan string]bool{}
var sseMu sync.Mutex

func autoInputHandler(w http.ResponseWriter, r *http.Request) {
    word := strings.ToLower(r.URL.Query().Get("q"))
    if word == "" {
        return
    }

    // Broadcast ke semua SSE client
    sseMu.Lock()
    for ch := range sseClients {
        select {
        case ch <- word:
        default:
        }
    }
    sseMu.Unlock()

    w.WriteHeader(http.StatusOK)
}

func sseHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("Access-Control-Allow-Origin", "*")

    ch := make(chan string, 5)
    sseMu.Lock()
    sseClients[ch] = true
    sseMu.Unlock()

    defer func() {
        sseMu.Lock()
        delete(sseClients, ch)
        sseMu.Unlock()
    }()

    for {
        select {
        case word := <-ch:
            fmt.Fprintf(w, "data: %s\n\n", word)
            w.(http.Flusher).Flush()
        case <-r.Context().Done():
            return
        }
    }
}

func loadDeleted() {
    data, err := os.ReadFile(deletedFile)
    if err != nil {
        return
    }
    json.Unmarshal(data, &deletedWords)
    fmt.Printf("Deleted words dimuat: %d kata\n", len(deletedWords))

    // Buat set untuk lookup cepat
    deletedSet := map[string]bool{}
    for _, w := range deletedWords {
        deletedSet[w] = true
    }

    // Buang kata yang sudah dihapus dari words
    filtered := words[:0]
    for _, w := range words {
        if !deletedSet[w] {
            filtered = append(filtered, w)
        }
    }
    words = filtered

    fmt.Printf("Words setelah filter: %d kata\n", len(words))
}

func saveDeleted() {
    data, _ := json.MarshalIndent(deletedWords, "", "  ")
    os.WriteFile(deletedFile, data, 0644)
}

func deleteWordHandler(w http.ResponseWriter, r *http.Request) {
    word := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
    if word == "" {
        http.Error(w, "query kosong", 400)
        return
    }

    deletedMu.Lock()
    defer deletedMu.Unlock()

    // Hapus dari words
    newWords := words[:0]
    found := false
    for _, ww := range words {
        if ww == word {
            found = true
        } else {
            newWords = append(newWords, ww)
        }
    }
    if !found {
        http.Error(w, "kata tidak ditemukan", 404)
        return
    }
    words = newWords

    // Simpan ke deleted list
    deletedWords = append(deletedWords, word)
    saveDeleted()

    // Rebuild index
    prefixIndex = map[string][]string{}
    suffixIndex = map[string][]string{}
    prefixFrequency = map[string]int{}
    deadEndSuffixes = map[string]bool{}
    buildIndex()
    buildSmartIndex()

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]any{"status": "deleted", "word": word})
}

// Handler baru: kembalikan semua key dari killerSuffix beserta skornya
func killerSuffixHandler(w http.ResponseWriter, r *http.Request) {
    type SuffixItem struct {
        Suffix string `json:"suffix"`
        Score  int    `json:"score"`
    }

    items := make([]SuffixItem, 0, len(killerSuffix))
    for suffix, score := range killerSuffix {
        items = append(items, SuffixItem{Suffix: suffix, Score: score})
    }

    // Sort by score descending (yang paling "mematikan" di atas)
    sort.Slice(items, func(i, j int) bool {
        return items[i].Score > items[j].Score
    })

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(items)
}

func suggestedSuffixDataHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(suggestedSuffixes)
}

func main() {
    // 1. Load data dulu
    loadKamus()
	loadDeleted() 
	loadSuggestedSuffixes()
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

	http.HandleFunc("/launcher.html", func(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "./templates/launcher.html")
	})
	http.HandleFunc("/widget2.html", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./templates/widget2.html")
	})

	// API
	http.HandleFunc("/search", searchHandler)
	http.HandleFunc("/search2",  searchHandlerV2)
	http.HandleFunc("/auto-input", autoInputHandler)
	http.HandleFunc("/sse", sseHandler)
	http.HandleFunc("/delete-word", deleteWordHandler)  
	http.HandleFunc("/danger-words", dangerWordsHandler)
	http.HandleFunc("/killer-suffix", killerSuffixHandler)
	http.HandleFunc("/suggested-suffix", suggestedSuffixHandler)
	http.HandleFunc("/delete-suggested-suffix", deleteSuggestedSuffixHandler)
	http.HandleFunc("/rekomendasi", func(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "./templates/suggested-analysis.html")
	})
	http.HandleFunc("/api/suggested-suffix-data", suggestedSuffixDataHandler)

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