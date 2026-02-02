package words

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/kljensen/snowball"
	"github.com/kljensen/snowball/english"
)

var availableCharacters = regexp.MustCompile("[A-Za-z0-9]+")

func isDigits(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func Normalize(phrase string) []string {

	if phrase == "" {
		return []string{}
	}

	raw := availableCharacters.FindAllString(phrase, -1)

	if len(raw) == 0 {
		return []string{}
	}

	out := make([]string, 0, len(raw))

	// в тестах проверка на дубликаты
	seen := make(map[string]bool) // Повышаем читаемость

	for _, word := range raw {

		// Стоп слова рассчитаны на lowerCase
		// "The" не фильтруется как стоп слово, алгоритм пропустит, поэтому так
		w := strings.ToLower(word)

		// числа пропускаем, в задании ничего
		if isDigits(w) {
			if !seen[w] {
				out = append(out, w)
				seen[w] = true
			}
			continue
		}

		// отсеивание of/a/the/
		if english.IsStopWord(w) {
			continue
		}

		stem, err := snowball.Stem(w, "english", true)
		if err != nil && stem == "" {
			stem = w // fallback к исходному слову
		}

		if !seen[stem] {
			out = append(out, stem)
			seen[stem] = true
		}
	}
	return out
}
