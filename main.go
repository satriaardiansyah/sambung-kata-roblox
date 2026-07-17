package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
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

// var killerSuffix = map[string]int{
// 	// 5 Karakter
// 	"orasi": 1000,
// 	"kisme": 1000,
// 	"sitas": 950,
// 	"takan": 950,
// 	"iskan": 950,
// 	"likan": 950,
// 	"litik": 950,
// 	"aksis": 900,
// 	"angsa": 1000,
// 	"ering": 400,
// 	"rodok": 1000,
// 	"inggi": 900,
// 	"abaka": 999,
// 	"stele": 999,
// 	"alari": 950,
// 	"anasi": 950,
// 	"entil": 950,
// 	"anser": 900,
// 	"nggor": 900,
// 	"tipus": 900,
// 	"ancar": 800,
// 	"andur": 800,
// 	"angus": 800,
// 	"ansor": 800,
// 	"antem": 800,
// 	"arong": 800,
// 	"arkil": 800,
// 	"awang": 800,
// 	"elli":  800,
// 	"ensil": 800,
// 	"ergot": 800,
// 	"hapak": 800,
// 	"kimah": 800,
// 	"lahad": 800,
// 	"latah": 800,
// 	"matis": 900,
// 	"ofoni": 800,
// 	"oleac": 800,
// 	"olong": 800,
// 	"ritis": 800,
// 	"tanai": 800,
// 	"tisis": 800,
// 	"tonik": 800,
// 	"ungsi": 800,
// 	"ahang": 580,
// 	"fauna": 580,
// 	"garot": 580,
// 	"gatot": 580,
// 	"mboli": 580,
// 	"ngudo": 580,
// 	"olang": 580,
// 	"sosro": 580,
// 	"amang": 550,
// 	"hohon": 550,
// 	"isian": 550,
// 	"riksa": 550,
// 	"trium": 550,
// 	"hiran": 500,
// 	"ganas": 450,
// 	"garpu": 450,
// 	"jijik": 450,
// 	"kolam": 450,
// 	"manat": 450,
// 	"meula": 450,
// 	"nusuk": 450,
// 	"ratif": 450,
// 	"umang": 450,
// 	"burma": 400,
// 	"fault": 400,
// 	"ruang": 400,
// 	"tikam": 400,
// 	"duksi": 300,
// 	"genik": 300,
// 	"kanya": 300,
// 	"logis": 300,
// 	"nggar": 300,
// 	"arian": 270,
// 	"meter": 100,
// 	"ogram": 100,

// 	// 4 Karakter
// 	"tusa": 800,
// 	"eran": 800,
// 	"atik": 800,
// 	"rian": 800,
// 	"iran": 550,
// 	"alah": 800,
// 	"inkan": 400,
// 	"taat":  400,
// 	"tiol":  350,
// 	"ilok":  320,
// 	"lipe":  320,
// 	"anki":  300,
// 	"atat":  300,
// 	"epik":  300,
// 	"inggu": 300,
// 	"ngeh":  300,
// 	"ngoh":  300,
// 	"riko":  300,
// 	"stis":  300,
// 	"wati":  300,
// 	"anah":  250,
// 	"asel":  250,
// 	"apet":  200,

// 	// 3 Karakter
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

var killerSuffix = map[string]int{
	// 5 Karakter (dari suggested_suffixes.json, 357 entri)
	// sort: prefix_count ASC, hits DESC
	"abaka" : 1000, // prefix_count=1, hits=12
	"ratif" : 1000, // prefix_count=1, hits=10
	"litik" :  990, // prefix_count=2, hits=45
	"ungsi" :  990, // prefix_count=2, hits=43
	"likan" :  990, // prefix_count=2, hits=34
	"angus" :  990, // prefix_count=2, hits=32
	"nggor" :  700, // prefix_count=2, hits=29
	"rodok" :  980, // prefix_count=2, hits=26
	"ritis" :  980, // prefix_count=2, hits=24
	"ngihu" :  980, // prefix_count=2, hits=19
	"ansor" :  980, // prefix_count=2, hits=18
	"kisme" :  970, // prefix_count=2, hits=18
	"tisis" :  970, // prefix_count=2, hits=18
	"anser" :  970, // prefix_count=2, hits=17
	"stele" :  970, // prefix_count=2, hits=17
	"alari" :  960, // prefix_count=2, hits=16
	"konis" :  960, // prefix_count=2, hits=14
	"riksa" :  960, // prefix_count=2, hits=13
	"tonik" :  960, // prefix_count=2, hits=13
	"usang" :  960, // prefix_count=2, hits=13
	"antri" :  950, // prefix_count=2, hits=12
	"erang" :  950, // prefix_count=2, hits=12
	"iskan" :  950, // prefix_count=2, hits=12
	"ancar" :  950, // prefix_count=2, hits=11
	"arung" :  940, // prefix_count=2, hits=11
	"tikam" :  940, // prefix_count=2, hits=11
	"andur" :  940, // prefix_count=2, hits=10
	"fauna" :  940, // prefix_count=2, hits=10
	"kanya" :  930, // prefix_count=2, hits=10
	"warsa" :  930, // prefix_count=2, hits=10
	"arong" :  930, // prefix_count=2, hits=9
	"awang" :  930, // prefix_count=2, hits=9
	"siden" :  930, // prefix_count=2, hits=9
	"ahana" :  920, // prefix_count=2, hits=8
	"anjar" :  920, // prefix_count=2, hits=8
	"ating" :  920, // prefix_count=2, hits=8
	"ikada" :  920, // prefix_count=2, hits=8
	"ongah" :  910, // prefix_count=2, hits=8
	"tikal" :  910, // prefix_count=2, hits=8
	"entil" :  910, // prefix_count=2, hits=7
	"isian" :  910, // prefix_count=2, hits=7
	"siong" :  910, // prefix_count=2, hits=7
	"tikus" :  900, // prefix_count=2, hits=7
	"versi" :  900, // prefix_count=2, hits=7
	"angai" :  900, // prefix_count=2, hits=6
	"angon" :  900, // prefix_count=2, hits=6
	"ensil" :  890, // prefix_count=2, hits=6
	"irani" :  890, // prefix_count=2, hits=6
	"niaga" :  890, // prefix_count=2, hits=6
	"orasi" :  890, // prefix_count=2, hits=6
	"tisik" :  890, // prefix_count=2, hits=6
	"umang" :  880, // prefix_count=2, hits=6
	"urang" :  880, // prefix_count=2, hits=6
	"gatot" :  880, // prefix_count=2, hits=5
	"lakah" :  880, // prefix_count=2, hits=5
	"lapet" :  870, // prefix_count=2, hits=5
	"latah" :  870, // prefix_count=2, hits=5
	"nahak" :  870, // prefix_count=2, hits=5
	"tasik" :  870, // prefix_count=2, hits=5
	"tikai" :  870, // prefix_count=2, hits=5
	"ambua" :  860, // prefix_count=2, hits=4
	"antek" :  860, // prefix_count=2, hits=4
	"hanat" :  860, // prefix_count=2, hits=4
	"hohon" :  860, // prefix_count=2, hits=4
	"osofi" :  850, // prefix_count=2, hits=4
	"tatik" :  850, // prefix_count=2, hits=4
	"tikah" :  850, // prefix_count=2, hits=4
	"eksin" :  850, // prefix_count=2, hits=3
	"garpu" :  840, // prefix_count=2, hits=3
	"gilik" :  840, // prefix_count=2, hits=3
	"lasem" :  840, // prefix_count=2, hits=3
	"nyaru" :  840, // prefix_count=2, hits=3
	"pisau" :  840, // prefix_count=2, hits=3
	"punan" :  830, // prefix_count=2, hits=3
	"rafia" :  830, // prefix_count=2, hits=3
	"ricau" :  830, // prefix_count=2, hits=3
	"rihat" :  830, // prefix_count=2, hits=3
	"risau" :  820, // prefix_count=2, hits=3
	"siria" :  820, // prefix_count=2, hits=3
	"tesis" :  820, // prefix_count=2, hits=3
	"tombe" :  820, // prefix_count=2, hits=3
	"ulung" :  820, // prefix_count=2, hits=3
	"umpat" :  810, // prefix_count=2, hits=3
	"ambak" :  810, // prefix_count=2, hits=2
	"arama" :  810, // prefix_count=2, hits=2
	"areng" :  810, // prefix_count=2, hits=2
	"atang" :  800, // prefix_count=2, hits=2
	"ayang" :  800, // prefix_count=2, hits=2
	"bilik" :  800, // prefix_count=2, hits=2
	"duren" :  800, // prefix_count=2, hits=2
	"emang" :  800, // prefix_count=2, hits=2
	"gangi" :  790, // prefix_count=2, hits=2
	"gilas" :  790, // prefix_count=2, hits=2
	"gitik" :  790, // prefix_count=2, hits=2
	"gores" :  790, // prefix_count=2, hits=2
	"kaiba" :  780, // prefix_count=2, hits=2
	"kasia" :  780, // prefix_count=2, hits=2
	"kusut" :  780, // prefix_count=2, hits=2
	"lalap" :  780, // prefix_count=2, hits=2
	"manat" :  780, // prefix_count=2, hits=2
	"medis" :  770, // prefix_count=2, hits=2
	"murni" :  770, // prefix_count=2, hits=2
	"ngaku" :  770, // prefix_count=2, hits=2
	"ngana" :  770, // prefix_count=2, hits=2
	"nging" :  760, // prefix_count=2, hits=2
	"pilot" :  760, // prefix_count=2, hits=2
	"ramah" :  760, // prefix_count=2, hits=2
	"rapuh" :  760, // prefix_count=2, hits=2
	"ratak" :  760, // prefix_count=2, hits=2
	"reksa" :  750, // prefix_count=2, hits=2
	"rinya" :  750, // prefix_count=2, hits=2
	"rokan" :  750, // prefix_count=2, hits=2
	"rosok" :  750, // prefix_count=2, hits=2
	"ruang" :  740, // prefix_count=2, hits=2
	"silih" :  740, // prefix_count=2, hits=2
	"siung" :  740, // prefix_count=2, hits=2
	"sogok" :  740, // prefix_count=2, hits=2
	"sokan" :  730, // prefix_count=2, hits=2
	"talat" :  730, // prefix_count=2, hits=2
	"teduh" :  730, // prefix_count=2, hits=2
	"tikel" :  730, // prefix_count=2, hits=2
	"tiket" :  730, // prefix_count=2, hits=2
	"troli" :  720, // prefix_count=2, hits=2
	"udang" :  720, // prefix_count=2, hits=2
	"ursus" :  720, // prefix_count=2, hits=2
	"afiks" :  720, // prefix_count=2, hits=1
	"akasa" :  710, // prefix_count=2, hits=1
	"aksos" :  710, // prefix_count=2, hits=1
	"akuik" :  710, // prefix_count=2, hits=1
	"anani" :  710, // prefix_count=2, hits=1
	"anker" :  710, // prefix_count=2, hits=1
	"ansar" :  700, // prefix_count=2, hits=1
	"apang" :  700, // prefix_count=2, hits=1
	"arane" :  700, // prefix_count=2, hits=1
	"atasi" :  700, // prefix_count=2, hits=1
	"biala" :  690, // prefix_count=2, hits=1
	"bokan" :  690, // prefix_count=2, hits=1
	"ektro" :  690, // prefix_count=2, hits=1
	"envoi" :  690, // prefix_count=2, hits=1
	"fasih" :  690, // prefix_count=2, hits=1
	"ferin" :  680, // prefix_count=2, hits=1
	"fluks" :  680, // prefix_count=2, hits=1
	"fokus" :  680, // prefix_count=2, hits=1
	"genan" :  680, // prefix_count=2, hits=1
	"getah" :  670, // prefix_count=2, hits=1
	"gusur" :  670, // prefix_count=2, hits=1
	"hasut" :  670, // prefix_count=2, hits=1
	"idola" :  670, // prefix_count=2, hits=1
	"ihsan" :  670, // prefix_count=2, hits=1
	"kimah" :  660, // prefix_count=2, hits=1
	"kulah" :  660, // prefix_count=2, hits=1
	"lenan" :  660, // prefix_count=2, hits=1
	"lensa" :  660, // prefix_count=2, hits=1
	"lepak" :  650, // prefix_count=2, hits=1
	"lirih" :  650, // prefix_count=2, hits=1
	"lokan" :  650, // prefix_count=2, hits=1
	"luruh" :  650, // prefix_count=2, hits=1
	"malah" :  640, // prefix_count=2, hits=1
	"marus" :  640, // prefix_count=2, hits=1
	"mesat" :  640, // prefix_count=2, hits=1
	"mesra" :  640, // prefix_count=2, hits=1
	"milih" :  640, // prefix_count=2, hits=1
	"muran" :  630, // prefix_count=2, hits=1
	"najis" :  630, // prefix_count=2, hits=1
	"ngani" :  630, // prefix_count=2, hits=1
	"nisan" :  630, // prefix_count=2, hits=1
	"ofoni" :  620, // prefix_count=2, hits=1
	"oksin" :  620, // prefix_count=2, hits=1
	"olang" :  620, // prefix_count=2, hits=1
	"ongga" :  620, // prefix_count=2, hits=1
	"opini" :  620, // prefix_count=2, hits=1
	"rapia" :  610, // prefix_count=2, hits=1
	"ratah" :  610, // prefix_count=2, hits=1
	"rusuk" :  610, // prefix_count=2, hits=1
	"sanad" :  610, // prefix_count=2, hits=1
	"sirip" :  600, // prefix_count=2, hits=1
	"solek" :  600, // prefix_count=2, hits=1
	"swana" :  600, // prefix_count=2, hits=1
	"talia" :  600, // prefix_count=2, hits=1
	"tania" :  600, // prefix_count=2, hits=1
	"taraf" :  590, // prefix_count=2, hits=1
	"teken" :  590, // prefix_count=2, hits=1
	"totok" :  590, // prefix_count=2, hits=1
	"tunai" :  590, // prefix_count=2, hits=1
	"tusuk" :  580, // prefix_count=2, hits=1
	"usung" :  580, // prefix_count=2, hits=1
	"inggi" :  580, // prefix_count=3, hits=79
	"matis" :  900, // prefix_count=3, hits=71
	"sitas" :  900, // prefix_count=3, hits=41
	"lahan" :  570, // prefix_count=3, hits=23
	"folia" :  570, // prefix_count=3, hits=21
	"anasi" :  570, // prefix_count=3, hits=20
	"burma" :  570, // prefix_count=3, hits=16
	"liang" :  560, // prefix_count=3, hits=15
	"trium" :  560, // prefix_count=3, hits=15
	"siksa" :  560, // prefix_count=3, hits=12
	"olong" :  560, // prefix_count=3, hits=10
	"ilang" :  560, // prefix_count=3, hits=9
	"takan" :  550, // prefix_count=3, hits=9
	"ahang" :  550, // prefix_count=3, hits=8
	"angat" :  550, // prefix_count=3, hits=8
	"lahat" :  550, // prefix_count=3, hits=8
	"gular" :  540, // prefix_count=3, hits=7
	"ongka" :  540, // prefix_count=3, hits=6
	"sakai" :  540, // prefix_count=3, hits=6
	"nanah" :  540, // prefix_count=3, hits=5
	"nikel" :  530, // prefix_count=3, hits=5
	"toris" :  530, // prefix_count=3, hits=5
	"angen" :  530, // prefix_count=3, hits=4
	"anyam" :  530, // prefix_count=3, hits=4
	"apala" :  530, // prefix_count=3, hits=4
	"lahap" :  520, // prefix_count=3, hits=4
	"pesek" :  520, // prefix_count=3, hits=4
	"talah" :  520, // prefix_count=3, hits=4
	"ambel" :  520, // prefix_count=3, hits=3
	"deran" :  510, // prefix_count=3, hits=3
	"etika" :  510, // prefix_count=3, hits=3
	"fitri" :  510, // prefix_count=3, hits=3
	"ganga" :  510, // prefix_count=3, hits=3
	"iring" :  510, // prefix_count=3, hits=3
	"kadin" :  500, // prefix_count=3, hits=3
	"lakan" :  500, // prefix_count=3, hits=3
	"larik" :  500, // prefix_count=3, hits=3
	"sekta" :  500, // prefix_count=3, hits=3
	"siasi" :  490, // prefix_count=3, hits=3
	"adang" :  490, // prefix_count=3, hits=2
	"aktan" :  490, // prefix_count=3, hits=2
	"alogi" :  490, // prefix_count=3, hits=2
	"antus" :  490, // prefix_count=3, hits=2
	"batik" :  480, // prefix_count=3, hits=2
	"bungo" :  480, // prefix_count=3, hits=2
	"hasil" :  480, // prefix_count=3, hits=2
	"iling" :  480, // prefix_count=3, hits=2
	"imbal" :  470, // prefix_count=3, hits=2
	"kadir" :  470, // prefix_count=3, hits=2
	"laria" :  470, // prefix_count=3, hits=2
	"lawat" :  470, // prefix_count=3, hits=2
	"leusi" :  470, // prefix_count=3, hits=2
	"niran" :  460, // prefix_count=3, hits=2
	"piang" :  460, // prefix_count=3, hits=2
	"rasis" :  460, // prefix_count=3, hits=2
	"sarah" :  460, // prefix_count=3, hits=2
	"taria" :  450, // prefix_count=3, hits=2
	"upaya" :  450, // prefix_count=3, hits=2
	"ampan" :  450, // prefix_count=3, hits=1
	"amuka" :  450, // prefix_count=3, hits=1
	"anduk" :  440, // prefix_count=3, hits=1
	"arami" :  440, // prefix_count=3, hits=1
	"atala" :  440, // prefix_count=3, hits=1
	"atori" :  440, // prefix_count=3, hits=1
	"aurat" :  440, // prefix_count=3, hits=1
	"danan" :  430, // prefix_count=3, hits=1
	"darah" :  430, // prefix_count=3, hits=1
	"darus" :  430, // prefix_count=3, hits=1
	"endar" :  430, // prefix_count=3, hits=1
	"gapit" :  420, // prefix_count=3, hits=1
	"garas" :  420, // prefix_count=3, hits=1
	"gaung" :  420, // prefix_count=3, hits=1
	"ingus" :  420, // prefix_count=3, hits=1
	"kerit" :  420, // prefix_count=3, hits=1
	"kitab" :  410, // prefix_count=3, hits=1
	"krama" :  410, // prefix_count=3, hits=1
	"lasia" :  410, // prefix_count=3, hits=1
	"leher" :  410, // prefix_count=3, hits=1
	"lesit" :  400, // prefix_count=3, hits=1
	"luang" :  400, // prefix_count=3, hits=1
	"lusif" :  400, // prefix_count=3, hits=1
	"makin" :  400, // prefix_count=3, hits=1
	"mamak" :  400, // prefix_count=3, hits=1
	"mutan" :  390, // prefix_count=3, hits=1
	"oksia" :  390, // prefix_count=3, hits=1
	"patis" :  390, // prefix_count=3, hits=1
	"rango" :  390, // prefix_count=3, hits=1
	"ratus" :  380, // prefix_count=3, hits=1
	"rungi" :  380, // prefix_count=3, hits=1
	"sarok" :  380, // prefix_count=3, hits=1
	"simis" :  380, // prefix_count=3, hits=1
	"sisih" :  380, // prefix_count=3, hits=1
	"sitin" :  370, // prefix_count=3, hits=1
	"sukan" :  370, // prefix_count=3, hits=1
	"talik" :  370, // prefix_count=3, hits=1
	"teler" :  370, // prefix_count=3, hits=1
	"tetep" :  360, // prefix_count=3, hits=1
	"turan" :  360, // prefix_count=3, hits=1
	"wahan" :  360, // prefix_count=3, hits=1
	"angsa" :  360, // prefix_count=4, hits=329
	"arian" :  360, // prefix_count=4, hits=49
	"aksis" :  350, // prefix_count=4, hits=20
	"biang" :  350, // prefix_count=4, hits=9
	"inggu" :  350, // prefix_count=4, hits=6
	"kiran" :  350, // prefix_count=4, hits=6
	"fiksi" :  340, // prefix_count=4, hits=5
	"nikah" :  340, // prefix_count=4, hits=5
	"inkan" :  340, // prefix_count=4, hits=4
	"sinya" :  340, // prefix_count=4, hits=4
	"palah" :  330, // prefix_count=4, hits=3
	"sofis" :  330, // prefix_count=4, hits=3
	"strip" :  330, // prefix_count=4, hits=3
	"akari" :  330, // prefix_count=4, hits=2
	"amang" :  330, // prefix_count=4, hits=2
	"ambat" :  320, // prefix_count=4, hits=2
	"antak" :  320, // prefix_count=4, hits=2
	"gawan" :  320, // prefix_count=4, hits=2
	"giran" :  320, // prefix_count=4, hits=2
	"gitar" :  310, // prefix_count=4, hits=2
	"guani" :  310, // prefix_count=4, hits=2
	"hilir" :  310, // prefix_count=4, hits=2
	"nanti" :  310, // prefix_count=4, hits=2
	"nguru" :  310, // prefix_count=4, hits=2
	"ograf" :  300, // prefix_count=4, hits=2
	"oksik" :  300, // prefix_count=4, hits=2
	"omong" :  300, // prefix_count=4, hits=2
	"recht" :  300, // prefix_count=4, hits=2
	"sasak" :  290, // prefix_count=4, hits=2
	"strik" :  290, // prefix_count=4, hits=2
	"talen" :  290, // prefix_count=4, hits=2
	"adina" :  290, // prefix_count=4, hits=1
	"alias" :  290, // prefix_count=4, hits=1
	"andak" :  280, // prefix_count=4, hits=1
	"atlet" :  280, // prefix_count=4, hits=1
	"kulum" :  280, // prefix_count=4, hits=1
	"kusan" :  280, // prefix_count=4, hits=1
	"lalah" :  270, // prefix_count=4, hits=1
	"raung" :  270, // prefix_count=4, hits=1
	"istis" :  270, // prefix_count=5, hits=18
	"trian" :  270, // prefix_count=5, hits=14
	"aktif" :  270, // prefix_count=5, hits=5
	"andar" :  260, // prefix_count=5, hits=5
	"siran" :  260, // prefix_count=5, hits=5
	"matik" :  260, // prefix_count=5, hits=4
	"duksi" :  260, // prefix_count=5, hits=3
	"kanik" :  250, // prefix_count=5, hits=3
	"mania" :  250, // prefix_count=5, hits=3
	"asing" :  250, // prefix_count=5, hits=2
	"balah" :  250, // prefix_count=5, hits=2
	"batis" :  240, // prefix_count=5, hits=2
	"belar" :  240, // prefix_count=5, hits=2
	"densi" :  240, // prefix_count=5, hits=2
	"dikit" :  240, // prefix_count=5, hits=2
	"garap" :  240, // prefix_count=5, hits=2
	"kakak" :  230, // prefix_count=5, hits=2
	"kilan" :  230, // prefix_count=5, hits=2
	"latif" :  230, // prefix_count=5, hits=2
	"lewat" :  230, // prefix_count=5, hits=2
	"ngeng" :  220, // prefix_count=5, hits=2
	"ngong" :  220, // prefix_count=5, hits=2
	"rekan" :  220, // prefix_count=5, hits=2
	"sinol" :  220, // prefix_count=5, hits=2
	"sulan" :  220, // prefix_count=5, hits=2
	"dakwa" :  210, // prefix_count=5, hits=1
	"gelar" :  210, // prefix_count=5, hits=1
	"habis" :  210, // prefix_count=5, hits=1
	"haram" :  210, // prefix_count=5, hits=1
	"istik" :  200, // prefix_count=5, hits=1
	"kacau" :  200, // prefix_count=5, hits=1
	"lahir" :  200, // prefix_count=5, hits=1

	// 4 Karakter (dari suggested_suffixes.json, 225 entri)
	// sort: prefix_count ASC, hits DESC
	"nggi"  :  800, // prefix_count=1, hits=1
	"unan"  :  800, // prefix_count=1, hits=1
	"yata"  :  790, // prefix_count=1, hits=1
	"erin"  :  790, // prefix_count=2, hits=1
	"ilai"  :  790, // prefix_count=2, hits=1
	"ngsa"  :  790, // prefix_count=2, hits=1
	"rian"  :  780, // prefix_count=3, hits=155
	"alah"  :  780, // prefix_count=3, hits=117
	"sori"  :  780, // prefix_count=3, hits=29
	"sihi"  :  770, // prefix_count=3, hits=26
	"liar"  :  770, // prefix_count=3, hits=21
	"akai"  :  770, // prefix_count=3, hits=19
	"anik"  :  770, // prefix_count=3, hits=16
	"eran"  :  760, // prefix_count=3, hits=16
	"siur"  :  760, // prefix_count=3, hits=13
	"siat"  :  760, // prefix_count=3, hits=9
	"ahir"  :  750, // prefix_count=3, hits=8
	"rica"  :  750, // prefix_count=3, hits=7
	"usan"  :  750, // prefix_count=3, hits=7
	"asel"  :  740, // prefix_count=3, hits=6
	"ayab"  :  740, // prefix_count=3, hits=6
	"hiat"  :  740, // prefix_count=3, hits=6
	"onan"  :  740, // prefix_count=3, hits=6
	"usti"  :  730, // prefix_count=3, hits=6
	"jurk"  :  730, // prefix_count=3, hits=5
	"yala"  :  730, // prefix_count=3, hits=5
	"daik"  :  720, // prefix_count=3, hits=4
	"eusi"  :  720, // prefix_count=3, hits=4
	"ngoh"  :  720, // prefix_count=3, hits=4
	"osin"  :  720, // prefix_count=3, hits=4
	"ulat"  :  710, // prefix_count=3, hits=4
	"wahu"  :  710, // prefix_count=3, hits=4
	"apet"  :  710, // prefix_count=3, hits=3
	"asam"  :  700, // prefix_count=3, hits=3
	"azim"  :  700, // prefix_count=3, hits=3
	"biat"  :  700, // prefix_count=3, hits=3
	"gins"  :  700, // prefix_count=3, hits=3
	"huwa"  :  690, // prefix_count=3, hits=3
	"leha"  :  690, // prefix_count=3, hits=3
	"oceh"  :  690, // prefix_count=3, hits=3
	"orma"  :  680, // prefix_count=3, hits=3
	"owar"  :  680, // prefix_count=3, hits=3
	"rowa"  :  680, // prefix_count=3, hits=3
	"skar"  :  680, // prefix_count=3, hits=3
	"umum"  :  670, // prefix_count=3, hits=3
	"ural"  :  670, // prefix_count=3, hits=3
	"yasa"  :  670, // prefix_count=3, hits=3
	"abah"  :  660, // prefix_count=3, hits=2
	"adun"  :  660, // prefix_count=3, hits=2
	"agih"  :  660, // prefix_count=3, hits=2
	"akla"  :  650, // prefix_count=3, hits=2
	"amas"  :  650, // prefix_count=3, hits=2
	"apia"  :  650, // prefix_count=3, hits=2
	"aril"  :  650, // prefix_count=3, hits=2
	"brum"  :  640, // prefix_count=3, hits=2
	"ciki"  :  640, // prefix_count=3, hits=2
	"cuti"  :  640, // prefix_count=3, hits=2
	"elan"  :  630, // prefix_count=3, hits=2
	"elli"  :  630, // prefix_count=3, hits=2
	"eret"  :  630, // prefix_count=3, hits=2
	"fufu"  :  630, // prefix_count=3, hits=2
	"fumi"  :  620, // prefix_count=3, hits=2
	"gois"  :  620, // prefix_count=3, hits=2
	"idol"  :  620, // prefix_count=3, hits=2
	"ngok"  :  610, // prefix_count=3, hits=2
	"nyau"  :  610, // prefix_count=3, hits=2
	"ogan"  :  610, // prefix_count=3, hits=2
	"siul"  :  610, // prefix_count=3, hits=2
	"tret"  :  600, // prefix_count=3, hits=2
	"ukir"  :  600, // prefix_count=3, hits=2
	"waid"  :  600, // prefix_count=3, hits=2
	"yoha"  :  590, // prefix_count=3, hits=2
	"abur"  :  590, // prefix_count=3, hits=1
	"adeg"  :  590, // prefix_count=3, hits=1
	"adir"  :  590, // prefix_count=3, hits=1
	"akam"  :  580, // prefix_count=3, hits=1
	"alib"  :  580, // prefix_count=3, hits=1
	"aneh"  :  580, // prefix_count=3, hits=1
	"aruh"  :  570, // prefix_count=3, hits=1
	"asus"  :  570, // prefix_count=3, hits=1
	"atha"  :  570, // prefix_count=3, hits=1
	"awar"  :  560, // prefix_count=3, hits=1
	"awit"  :  560, // prefix_count=3, hits=1
	"ayak"  :  560, // prefix_count=3, hits=1
	"bori"  :  560, // prefix_count=3, hits=1
	"clea"  :  550, // prefix_count=3, hits=1
	"ekas"  :  550, // prefix_count=3, hits=1
	"elis"  :  550, // prefix_count=3, hits=1
	"enak"  :  540, // prefix_count=3, hits=1
	"enen"  :  540, // prefix_count=3, hits=1
	"gisa"  :  540, // prefix_count=3, hits=1
	"gour"  :  540, // prefix_count=3, hits=1
	"guta"  :  530, // prefix_count=3, hits=1
	"holu"  :  530, // prefix_count=3, hits=1
	"imsa"  :  530, // prefix_count=3, hits=1
	"nafi"  :  520, // prefix_count=3, hits=1
	"nalu"  :  520, // prefix_count=3, hits=1
	"nefo"  :  520, // prefix_count=3, hits=1
	"ofon"  :  520, // prefix_count=3, hits=1
	"okol"  :  510, // prefix_count=3, hits=1
	"omel"  :  510, // prefix_count=3, hits=1
	"oras"  :  510, // prefix_count=3, hits=1
	"puki"  :  500, // prefix_count=3, hits=1
	"raih"  :  500, // prefix_count=3, hits=1
	"reni"  :  500, // prefix_count=3, hits=1
	"sanu"  :  500, // prefix_count=3, hits=1
	"tiar"  :  490, // prefix_count=3, hits=1
	"tile"  :  490, // prefix_count=3, hits=1
	"tost"  :  490, // prefix_count=3, hits=1
	"troi"  :  480, // prefix_count=3, hits=1
	"ujar"  :  480, // prefix_count=3, hits=1
	"ulur"  :  480, // prefix_count=3, hits=1
	"urat"  :  480, // prefix_count=3, hits=1
	"usir"  :  470, // prefix_count=3, hits=1
	"yali"  :  470, // prefix_count=3, hits=1
	"zina"  :  470, // prefix_count=3, hits=1
	"wati"  :  460, // prefix_count=4, hits=60
	"ngeh"  :  460, // prefix_count=4, hits=37
	"atik"  :  460, // prefix_count=4, hits=30
	"stis"  :  450, // prefix_count=4, hits=27
	"onik"  :  450, // prefix_count=4, hits=16
	"ahad"  :  450, // prefix_count=4, hits=12
	"anje"  :  450, // prefix_count=4, hits=9
	"osit"  :  440, // prefix_count=4, hits=9
	"ikad"  :  440, // prefix_count=4, hits=8
	"klik"  :  440, // prefix_count=4, hits=8
	"anah"  :  430, // prefix_count=4, hits=7
	"neus"  :  430, // prefix_count=4, hits=7
	"tipe"  :  430, // prefix_count=4, hits=7
	"uang"  :  430, // prefix_count=4, hits=7
	"ilan"  :  420, // prefix_count=4, hits=6
	"deha"  :  420, // prefix_count=4, hits=5
	"niat"  :  420, // prefix_count=4, hits=5
	"orek"  :  410, // prefix_count=4, hits=5
	"alik"  :  410, // prefix_count=4, hits=4
	"arta"  :  410, // prefix_count=4, hits=4
	"usil"  :  410, // prefix_count=4, hits=4
	"anus"  :  400, // prefix_count=4, hits=3
	"afir"  :  400, // prefix_count=4, hits=2
	"alap"  :  400, // prefix_count=4, hits=2
	"alto"  :  390, // prefix_count=4, hits=2
	"amah"  :  390, // prefix_count=4, hits=2
	"apas"  :  390, // prefix_count=4, hits=2
	"asar"  :  390, // prefix_count=4, hits=2
	"asek"  :  380, // prefix_count=4, hits=2
	"asir"  :  380, // prefix_count=4, hits=2
	"atal"  :  380, // prefix_count=4, hits=2
	"diot"  :  370, // prefix_count=4, hits=2
	"frit"  :  370, // prefix_count=4, hits=2
	"ilok"  :  370, // prefix_count=4, hits=2
	"irah"  :  360, // prefix_count=4, hits=2
	"isak"  :  360, // prefix_count=4, hits=2
	"kait"  :  360, // prefix_count=4, hits=2
	"klab"  :  360, // prefix_count=4, hits=2
	"otop"  :  350, // prefix_count=4, hits=2
	"rias"  :  350, // prefix_count=4, hits=2
	"spid"  :  350, // prefix_count=4, hits=2
	"sual"  :  340, // prefix_count=4, hits=2
	"tiol"  :  340, // prefix_count=4, hits=2
	"tosa"  :  340, // prefix_count=4, hits=2
	"ukur"  :  340, // prefix_count=4, hits=2
	"abar"  :  330, // prefix_count=4, hits=1
	"abut"  :  330, // prefix_count=4, hits=1
	"asuh"  :  330, // prefix_count=4, hits=1
	"atar"  :  320, // prefix_count=4, hits=1
	"bito"  :  320, // prefix_count=4, hits=1
	"blat"  :  320, // prefix_count=4, hits=1
	"endi"  :  320, // prefix_count=4, hits=1
	"gawi"  :  310, // prefix_count=4, hits=1
	"kuil"  :  310, // prefix_count=4, hits=1
	"lade"  :  310, // prefix_count=4, hits=1
	"luks"  :  300, // prefix_count=4, hits=1
	"pler"  :  300, // prefix_count=4, hits=1
	"riko"  :  300, // prefix_count=4, hits=1
	"rote"  :  300, // prefix_count=4, hits=1
	"tain"  :  290, // prefix_count=4, hits=1
	"tsar"  :  290, // prefix_count=4, hits=1
	"itik"  :  290, // prefix_count=5, hits=50
	"iran"  :  280, // prefix_count=5, hits=44
	"ngus"  :  280, // prefix_count=5, hits=35
	"ngih"  :  280, // prefix_count=5, hits=14
	"tase"  :  270, // prefix_count=5, hits=12
	"atam"  :  270, // prefix_count=5, hits=8
	"gian"  :  270, // prefix_count=5, hits=6
	"roso"  :  270, // prefix_count=5, hits=5
	"adar"  :  260, // prefix_count=5, hits=4
	"ilah"  :  260, // prefix_count=5, hits=4
	"oris"  :  260, // prefix_count=5, hits=4
	"yaba"  :  250, // prefix_count=5, hits=4
	"atur"  :  250, // prefix_count=5, hits=3
	"abla"  :  250, // prefix_count=5, hits=2
	"akta"  :  250, // prefix_count=5, hits=2
	"anof"  :  240, // prefix_count=5, hits=2
	"asak"  :  240, // prefix_count=5, hits=2
	"doke"  :  240, // prefix_count=5, hits=2
	"giat"  :  230, // prefix_count=5, hits=2
	"hilo"  :  230, // prefix_count=5, hits=2
	"kohi"  :  230, // prefix_count=5, hits=2
	"liki"  :  230, // prefix_count=5, hits=2
	"naik"  :  220, // prefix_count=5, hits=2
	"nazi"  :  220, // prefix_count=5, hits=2
	"oten"  :  220, // prefix_count=5, hits=2
	"poni"  :  210, // prefix_count=5, hits=2
	"rina"  :  210, // prefix_count=5, hits=2
	"rise"  :  210, // prefix_count=5, hits=2
	"rosi"  :  210, // prefix_count=5, hits=2
	"same"  :  200, // prefix_count=5, hits=2
	"sins"  :  200, // prefix_count=5, hits=2
	"skop"  :  200, // prefix_count=5, hits=2
	"sogo"  :  190, // prefix_count=5, hits=2
	"trol"  :  190, // prefix_count=5, hits=2
	"tual"  :  190, // prefix_count=5, hits=2
	"tusa"  :  180, // prefix_count=5, hits=2
	"abak"  :  180, // prefix_count=5, hits=1
	"abun"  :  180, // prefix_count=5, hits=1
	"ator"  :  180, // prefix_count=5, hits=1
	"awak"  :  170, // prefix_count=5, hits=1
	"diks"  :  170, // prefix_count=5, hits=1
	"emis"  :  170, // prefix_count=5, hits=1
	"loha"  :  160, // prefix_count=5, hits=1
	"mian"  :  160, // prefix_count=5, hits=1
	"nesa"  :  160, // prefix_count=5, hits=1
	"okso"  :  160, // prefix_count=5, hits=1
	"rita"  :  150, // prefix_count=5, hits=1
	"riya"  :  150, // prefix_count=5, hits=1

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

// ============================================================================
// TYPING PRACTICE MODULE (NEW FEATURE)
// ============================================================================

type TypingSuffix struct {
	Suffix string `json:"suffix"`
	Count  int    `json:"count"`
	Score  int    `json:"score"`
}

var typingSuffixes []TypingSuffix

func buildTypingSuffixIndex() {
	suffixCount := map[string]int{}
	
	// Scan words to count matches for each killerSuffix key
	for _, w := range words {
		wClean := strings.ToLower(strings.TrimSpace(w))
		for suf := range killerSuffix {
			if strings.HasSuffix(wClean, suf) {
				suffixCount[suf]++
			}
		}
	}

	var list []TypingSuffix
	for suf, count := range suffixCount {
		if count > 0 {
			list = append(list, TypingSuffix{
				Suffix: suf,
				Count:  count,
				Score:  killerSuffix[suf],
			})
		}
	}

	// Sort by:
	// 1. score descending
	// 2. count descending
	// 3. suffix name alphabetically
	sort.Slice(list, func(i, j int) bool {
		if list[i].Score != list[j].Score {
			return list[i].Score > list[j].Score
		}
		if list[i].Count != list[j].Count {
			return list[i].Count > list[j].Count
		}
		return list[i].Suffix < list[j].Suffix
	})

	typingSuffixes = list
	fmt.Printf("Typing suffix index built: %d suffixes (sourced from killerSuffix)\n", len(typingSuffixes))
}

func apiTypingSuffixesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(typingSuffixes)
}

func apiTypingWordsHandler(w http.ResponseWriter, r *http.Request) {
	suffixesParam := r.URL.Query().Get("suffixes")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// If no suffixes are selected, enter "Semua Kata" (All Words) mode.
	// We return 300 random words from the entire dataset.
	if suffixesParam == "" {
		count := 300
		if len(words) < count {
			count = len(words)
		}
		
		randomWords := make([]string, 0, count)
		if len(words) > 0 {
			for i := 0; i < count; i++ {
				idx := rand.Intn(len(words))
				randomWords = append(randomWords, strings.ToLower(strings.TrimSpace(words[idx])))
			}
		}
		
		json.NewEncoder(w).Encode(map[string][]string{
			"": randomWords,
		})
		return
	}

	sufs := strings.Split(suffixesParam, ",")
	result := map[string][]string{}
	for _, s := range sufs {
		s = strings.ToLower(strings.TrimSpace(s))
		if s == "" {
			continue
		}
		result[s] = []string{}
	}

	for _, w := range words {
		wClean := strings.ToLower(strings.TrimSpace(w))
		for s := range result {
			if strings.HasSuffix(wClean, s) {
				result[s] = append(result[s], wClean)
			}
		}
	}

	json.NewEncoder(w).Encode(result)
}

func main() {
    // 1. Load data dulu
    loadKamus()
	loadDeleted() 
	loadSuggestedSuffixes()
    buildIndex()
	buildSmartIndex()
	buildTypingSuffixIndex()

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

	// Typing Practice Routes
	http.HandleFunc("/typing", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./templates/typing.html")
	})
	http.HandleFunc("/api/typing/suffixes", apiTypingSuffixesHandler)
	http.HandleFunc("/api/typing/words", apiTypingWordsHandler)

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