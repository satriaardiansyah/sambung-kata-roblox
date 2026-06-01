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

// var killerSuffix = map[string]int{
// 	"nggar": 500,
// 	"nggor": 500,
// 	"recth": 500,
// 	"logis": 600,
// 	"duksi": 580,
// 	"fauna": 580,
// 	"mboli": 580,
// 	"gatot": 580,
// 	"ahang": 580,
// 	"genik": 590,
// 	"riksa": 550,
// 	"iran": 550,
// 	"abaka": 999,
// 	"ngudo": 580,
// 	"sosro": 580,
// 	"garot": 580,
// 	"litik": 550,
// 	"olang": 580,
// 	"amang": 550,
// 	"trium": 550,
// 	"inkan": 400,
// 	"yala": 550,
// 	"aesi": 550,
// 	"hohon": 550,
// 	"isian": 550,
// 	"arkil": 500,
// 	"oleac": 500,
// 	"ensil": 500,
// 	"tanai": 500,
// 	"olong": 500,
// 	"hiran": 500,
// 	"ratif": 450,
// 	"nusuk": 450,
// 	"jijik": 450,
// 	"manat": 450,
// 	"meula": 450,
// 	"kolam": 450,
// 	"ganas": 450,
// 	"garpu": 450,
// 	"umang": 450,
// 	"abraka": 400,
// 	"tikam": 400,
// 	"taat": 400,
// 	"fault": 400,
// 	"burma": 400,
// 	"ruang": 400,
// 	"iat": 350,
// 	"tiol": 350,
// 	"ilok": 320,
// 	"lipe": 320,
// 	"ngeh": 300,
// 	"wati": 300,
// 	"riko": 300,
// 	"apet": 200,
// 	"inggu": 300,
// 	"alah": 300,
// 	"ngoh": 300,
// 	"anki": 300,
// 	"unc": 300,
// 	"stis": 300,
// 	"kanya": 300,
// 	"angus": 300,
// 	"ksa": 290,
// 	"nget": 270,
// 	"losa": 270,
// 	"eusi": 270,
// 	"ngih": 270,
// 	"awak": 270,
// 	"atat": 270,
// 	"arian": 270,
// 	"inggi": 460,
// 	"sih": 260,
// 	"hih": 260,
// 	"meh": 260,
// 	"owa": 260,
// 	"esi": 260,
// 	"huh": 250,
// 	"dot": 250,
// 	"pei": 250,
// 	"bou": 250,
// 	"pso": 250,
// 	"anah": 250,
// 	"asel": 250,
// 	"iya": 220,
// 	"yab": 200,
// 	"iki": 200,
// 	"ipe": 200,
// 	"voi": 200,
// 	"coe": 200,
// 	"ml": 260,
// 	"sm": 210,
// 	"iu": 200,
// 	"ez": 200,
// 	"ou": 200,
// 	"tl": 200,
// 	"moi": 200,
// 	"ks": 180,
// 	"gy": 170,
// 	"ox": 150,
// 	"oo": 140,
// 	"cy": 130,
// 	"ex": 120,
// 	"eo": 120,
// 	"eq": 120,
// 	"oi": 120,
// 	"pp": 100,
// 	"eh": 100,
// 	"x": 60,
// 	"z": 60,
// 	"q": 60,
// 	"w": 60,
// 	"c": 60,
// 	"F": 60,
// 	"V": 60,
// 	"ia": 60,
// 	"ng": 60,
// }


var killerSuffix = map[string]int{
	// 1 karakter
	"a": 60,
	"i": 60,
	"u": 60,
	"e": 60,
	"o": 60,
	"b": 80,
	"d": 80,
	"f": 80,
	"g": 80,
	"h": 80,
	"j": 80,
	"k": 80,
	"l": 80,
	"m": 80,
	"n": 80,
	"p": 80,
	"r": 80,
	"s": 80,
	"t": 80,

	// 2 karakter
	"ab": 100,
	"ac": 100,
	"ad": 100,
	"af": 100,
	"ag": 100,
	"ah": 100,
	"ai": 100,
	"aj": 100,
	"ak": 100,
	"al": 100,
	"am": 100,
	"an": 100,
	"ap": 100,
	"ar": 100,
	"as": 100,
	"at": 100,
	"au": 100,
	"av": 100,
	"aw": 100,
	"ay": 100,
	"ba": 120,
	"bi": 120,
	"bo": 120,
	"bu": 120,
	"da": 120,
	"di": 120,
	"du": 120,
	"ga": 120,
	"gi": 120,
	"gu": 120,
	"ha": 120,
	"hi": 120,
	"hu": 120,
	"ja": 120,
	"ji": 120,
	"ju": 120,
	"ka": 120,
	"ki": 120,
	"ku": 120,
	"la": 120,
	"li": 120,
	"lu": 120,
	"ma": 120,
	"mi": 120,
	"mu": 120,
	"na": 120,
	"ni": 120,
	"nu": 120,
	"pa": 120,
	"pi": 120,
	"pu": 120,
	"ra": 120,
	"ri": 120,
	"ru": 120,
	"sa": 120,
	"si": 120,
	"su": 120,
	"ta": 120,
	"ti": 120,
	"tu": 120,
	"wa": 120,
	"wi": 120,
	"wu": 120,
	"ya": 120,
	"yu": 120,

	// 3 karakter
	"aba": 150,
	"abi": 150,
	"abu": 150,
	"ada": 150,
	"adi": 150,
	"adu": 150,
	"aga": 150,
	"agi": 150,
	"agu": 150,
	"aha": 150,
	"ahi": 150,
	"ahu": 150,
	"aja": 150,
	"aji": 150,
	"aju": 150,
	"aka": 150,
	"aki": 150,
	"aku": 150,
	"ala": 150,
	"ali": 150,
	"alu": 150,
	"ama": 150,
	"ami": 150,
	"amu": 150,
	"ana": 150,
	"ani": 150,
	"anu": 150,
	"apa": 150,
	"api": 150,
	"apu": 150,
	"ara": 150,
	"ari": 150,
	"aru": 150,
	"asa": 150,
	"asi": 150,
	"asu": 150,
	"ata": 150,
	"ati": 150,
	"atu": 150,
	"aua": 150,
	"awa": 150,
	"awi": 150,
	"awu": 150,
	"aya": 150,
	"ayi": 150,
	"ayu": 150,
	"bal": 170,
	"ban": 170,
	"bar": 170,
	"bas": 170,
	"bat": 170,
	"bau": 170,
	"bil": 170,
	"bin": 170,
	"bir": 170,
	"bis": 170,
	"bit": 170,
	"bol": 170,
	"bon": 170,
	"bor": 170,
	"bos": 170,
	"bot": 170,
	"bul": 170,
	"bun": 170,
	"bur": 170,
	"bus": 170,
	"but": 170,
	"dal": 170,
	"dan": 170,
	"dar": 170,
	"das": 170,
	"dat": 170,
	"dil": 170,
	"din": 170,
	"dir": 170,
	"dis": 170,
	"dit": 170,
	"dol": 170,
	"don": 170,
	"dor": 170,
	"dos": 170,
	"dot": 170,
	"dul": 170,
	"dun": 170,
	"dur": 170,
	"dus": 170,
	"dut": 170,

	// 4 karakter
	"abdi": 580,
	"acak": 570,
	"aduk": 560,
	"agak": 550,
	"ajak": 540,
	"akad": 530,
	"akal": 520,
	"akan": 510,
	"akar": 500,
	"akas": 490,
	"akat": 480,
	"akil": 470,
	"akim": 460,
	"akin": 450,
	"akir": 440,
	"akis": 430,
	"akit": 420,
	"aksi": 410,
	"akta": 400,
	"aktu": 390,
	"alap": 380,
	"alar": 370,
	"alas": 360,
	"alat": 350,
	"alin": 340,
	"alir": 330,
	"alis": 320,
	"alit": 310,
	"alun": 300,
	"alur": 290,
	"amak": 280,
	"amal": 270,
	"aman": 260,
	"amar": 250,
	"amas": 240,
	"amat": 230,
	"ambi": 220,
	"ambu": 210,
	"amil": 200,
	"amin": 190,
	"amit": 180,
	"amuk": 170,
	"anas": 160,
	"anat": 150,
	"anda": 140,
	"andi": 130,
	"andu": 120,
	"aneh": 110,
	"anem": 100,
	"aner":  90,
	"anet":  80,
	"anga":  70,
	"angu":  60,
}

var killerOpener = map[string]int{
    "nggaro":    0,
	"inggih":   0,
    "genikulum": 0,
    "logistik":  0,
    "iranika":   0,
    "iranga":    0,
    "garpuan":   0,
	"olanggara": 0,
	"ahangkara": 0,
	"umangkapala": 0,
	"gatotkaca": 0,
	"tikaman": 0,
	"faunasia": 0,
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

func main() {
    // 1. Load data dulu
    loadKamus()
	loadDeleted() 
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
	http.HandleFunc("/auto-input", autoInputHandler)
	http.HandleFunc("/sse", sseHandler)
	http.HandleFunc("/delete-word", deleteWordHandler)  
	http.HandleFunc("/danger-words", dangerWordsHandler)
	http.HandleFunc("/killer-suffix", killerSuffixHandler)

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