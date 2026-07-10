package renamer

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// typos maps common misspellings (lowercase) to their corrections.
// Matching is case-insensitive on word boundaries and the correction
// preserves the case shape of the match (the→the, The→The, THE→THE).
var typos = map[string]string{
	"accommodate": "accommodate",
	"across":    "across",
	"and":        "and",
	"because":    "because",
	"believe":    "believe",
	"calendar":   "calendar",
	"could":      "could",
	"definitely": "definitely",
	"environment": "environment",
	"friend":     "friend",
	"the":        "the",
	"necessary": "necessary",
	"necessary":   "necessary",
	"occurred":    "occurred",
	"occurrence":  "occurrence",
	"publicly": "publicly",
	"receive":    "receive",
	"received":   "received",
	"recommend":  "recommend",
	"separate":   "separate",
	"should":     "should",
	"successful":  "successful",
	"successful":  "successful",
	"that":       "that",
	"the":        "the",
	"thee":       "the",
	"their":      "their",
	"tomorrow":   "tomorrow",
	"until":     "until",
	"what":       "what",
	"which":      "which",
	"which":       "which",
	"weird":      "weird",
	"would":      "would",
}

// Typos returns a Replacer that fixes the built-in list of common typos.
func Typos() Replacer {
	// Longer typos first so e.g. "thee" wins over "the".
	words := make([]string, 0, len(typos))
	for w := range typos {
		words = append(words, w)
	}
	sort.Slice(words, func(i, j int) bool {
		if len(words[i]) != len(words[j]) {
			return len(words[i]) > len(words[j])
		}
		return words[i] < words[j]
	})
	quoted := make([]string, len(words))
	for i, w := range words {
		quoted[i] = regexp.QuoteMeta(w)
	}
	re := regexp.MustCompile(`(?i)\b(` + strings.Join(quoted, "|") + `)\b`)
	return typoReplacer{re: re}
}

// TypoList returns the known typo→correction pairs, sorted by typo.
func TypoList() [][2]string {
	pairs := make([][2]string, 0, len(typos))
	for w, c := range typos {
		pairs = append(pairs, [2]string{w, c})
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i][0] < pairs[j][0] })
	return pairs
}

type typoReplacer struct {
	re *regexp.Regexp
}

func (t typoReplacer) Replace(content []byte) ([]byte, int) {
	count := 0
	updated := t.re.ReplaceAllFunc(content, func(match []byte) []byte {
		correction, ok := typos[strings.ToLower(string(match))]
		if !ok {
			return match
		}
		count++
		return []byte(matchCase(string(match), correction))
	})
	if count == 0 {
		return content, 0
	}
	return updated, count
}

// matchCase reshapes correction to follow the case pattern of match:
// all-upper stays all-upper, leading capital stays capitalised,
// anything else is returned as-is (lowercase).
func matchCase(match, correction string) string {
	runes := []rune(match)
	allUpper := true
	for _, r := range runes {
		if !unicode.IsUpper(r) {
			allUpper = false
			break
		}
	}
	if allUpper && len(runes) > 1 {
		return strings.ToUpper(correction)
	}
	if unicode.IsUpper(runes[0]) {
		cr := []rune(correction)
		cr[0] = unicode.ToUpper(cr[0])
		return string(cr)
	}
	return correction
}
