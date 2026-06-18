package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

type Result struct {
	Suffix string
	Count  int
	Word   string
}

func test() {

	data, err := os.ReadFile("kbbi_updated.json")
	if err != nil {
		panic(err)
	}

	var words []string
	err = json.Unmarshal(data, &words)
	if err != nil {
		panic(err)
	}

	// normalisasi
	for i := range words {
		words[i] = strings.ToLower(strings.TrimSpace(words[i]))
	}

	prefixCount := make(map[string]int)
	prefixWord := make(map[string]string)

	suffixCount := make(map[string]int)

	//-----------------------------------
	// hitung prefix unik
	//-----------------------------------

	for _, w := range words {

		n := len(w)

		for l := 3; l <= 5; l++ {

			if n < l {
				continue
			}

			p := w[:l]

			prefixCount[p]++

			if _, ok := prefixWord[p]; !ok {
				prefixWord[p] = w
			}
		}
	}

	//-----------------------------------
	// hitung suffix
	//-----------------------------------

	for _, w := range words {

		n := len(w)

		for l := 3; l <= 5; l++ {

			if n < l {
				continue
			}

			s := w[n-l:]

			suffixCount[s]++
		}
	}

	//-----------------------------------
	// pilih suffix yang prefixnya unik
	//-----------------------------------

	var result []Result

	for suffix, count := range suffixCount {

		if prefixCount[suffix] == 1 {

			result = append(result, Result{
				Suffix: suffix,
				Count:  count,
				Word:   prefixWord[suffix],
			})
		}
	}

	sort.Slice(result, func(i, j int) bool {

		if result[i].Count == result[j].Count {
			return result[i].Suffix < result[j].Suffix
		}

		return result[i].Count > result[j].Count
	})

	//-----------------------------------
	// output
	//-----------------------------------

	var sb strings.Builder

sb.WriteString("package data\n\n")
sb.WriteString("var KillerEnding = map[string]int{\n")

for _, r := range result {

	score := 300

	switch len(r.Suffix) {
	case 5:
		score = 900
	case 4:
		score = 700
	case 3:
		score = 500
	}

	sb.WriteString(
		fmt.Sprintf(
			"\t%q: %d, // suffix=%d prefix=%s\n",
			r.Suffix,
			score,
			r.Count,
			r.Word,
		),
	)
}

sb.WriteString("}\n")

err = os.WriteFile("killerEnding.go", []byte(sb.String()), 0644)
if err != nil {
	panic(err)
}

fmt.Println("Selesai! File tersimpan di killerEnding.go")
}