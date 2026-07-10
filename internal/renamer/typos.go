package renamer

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// typos maps common misspellings (lowercase) to their corrections.
// Matching is case-insensitive on word boundaries and the correction
// preserves the case shape of the match: all-caps stays all-caps and a
// leading capital is kept.
//
// The misspelled keys are deliberately split into concatenated fragments so
// that running `rename -typo` over this repository cannot "correct" its own
// table — that would collapse entries into duplicate map keys and break the
// build. Word-boundary matching never sees the full misspelling in a
// fragment, so this file is immune.
var typos = map[string]string{
	"acco" + "modate": "accommodate",
	"acc" + "ross":    "across",
	"a" + "dn":        "and",
	"bec" + "uase":    "because",
	"bel" + "eive":    "believe",
	"cal" + "ender":   "calendar",
	"cou" + "dl":      "could",
	"defin" + "ately": "definitely",
	"envi" + "roment": "environment",
	"fre" + "ind":     "friend",
	"h" + "te":        "the",
	"necc" + "essary": "necessary",
	"nec" + "esary":   "necessary",
	"occ" + "ured":    "occurred",
	"occu" + "rence":  "occurrence",
	"public" + "ally": "publicly",
	"rec" + "ieve":    "receive",
	"rec" + "ieved":   "received",
	"recc" + "omend":  "recommend",
	"sep" + "erate":   "separate",
	"sho" + "udl":     "should",
	"succ" + "esful":  "successful",
	"suc" + "essful":  "successful",
	"ta" + "ht":       "that",
	"te" + "h":        "the",
	"te" + "he":       "the",
	"th" + "ier":      "their",
	"tomm" + "orow":   "tomorrow",
	"unt" + "ill":     "until",
	"wa" + "ht":       "what",
	"wh" + "cih":      "which",
	"wi" + "ch":       "which",
	"wi" + "erd":      "weird",
	"wou" + "dl":      "would",
}

// Typos returns a Replacer that fixes the built-in list of common typos.
func Typos() Replacer {
	// Longer misspellings first, so the longest match wins when one
	// misspelling is a prefix of another.
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
