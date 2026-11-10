package game

import (
	"strings"
	"unicode"
)

// Salvage: a rover that finishes a mission without damage (no frame or mast
// ever touched the ground) gives back one extra part for the next mission. In
// an expedition the part goes into the kit; in a single mission its credits
// go to the next mission's budget.

// SalvagePart is the part an undamaged rover gives back: its most valuable
// part other than the Navcam mast (every rover needs its own mast). Ties go to
// the earlier entry in Parts, so the choice is stable.
func SalvagePart(d Design) (Kind, bool) {
	best, bestCost := Kind(""), 0
	for _, p := range Parts {
		if p.Kind == KMast {
			continue
		}
		for _, c := range d.Cells {
			if c.K == p.Kind && p.Cost > bestCost {
				best, bestCost = p.Kind, p.Cost
				break
			}
		}
	}
	return best, best != ""
}

func partName(k Kind) string {
	p, _ := PartOf(k)
	return p.Name
}

// Rover names.

// MaxNameLen is the longest rover name, in characters.
const MaxNameLen = 18

var (
	nameFirst = []string{"Plucky", "Dusty", "Rusty", "Stubborn", "Wobbly", "Lucky", "Grumpy", "Nimble", "Sleepy", "Brave", "Tiny", "Bold"}
	nameLast  = []string{"Mule", "Beetle", "Pebble", "Badger", "Comet", "Tortoise", "Crab", "Sprocket", "Goat", "Wren", "Scout", "Pioneer"}
)

// SuggestName is the n-th generated rover name. Steps through every pairing
// before repeating.
func SuggestName(n int) string {
	if n < 0 {
		n = -n
	}
	a := n % len(nameFirst)
	b := (n*5 + n/len(nameFirst)) % len(nameLast)
	return nameFirst[a] + " " + nameLast[b]
}

// CleanName trims a name to letters, digits, spaces and simple punctuation the
// game font can draw, collapses spaces and caps the length.
func CleanName(s string) string {
	var b strings.Builder
	n, space := 0, false
	for _, r := range strings.TrimSpace(s) {
		if n >= MaxNameLen {
			break
		}
		switch {
		case r == ' ' || r == '\t':
			if space || n == 0 {
				continue
			}
			space = true
			b.WriteRune(' ')
		case nameRune(r):
			space = false
			b.WriteRune(r)
		default:
			continue
		}
		n++
	}
	return strings.TrimSpace(b.String())
}

// nameRune reports whether r may appear in a rover name (besides spaces).
func nameRune(r rune) bool {
	return r < 128 && (unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune("-'.!", r))
}

var numerals = []string{"", "", "II", "III", "IV", "V", "VI", "VII", "VIII", "IX", "X"}

// SuccessorName names the rover sent out after one was lost: "Plucky Mule"
// becomes "Plucky Mule II", then "Plucky Mule III".
func SuccessorName(name string) string {
	if name == "" {
		return ""
	}
	base, n := name, 1
	if i := strings.LastIndexByte(name, ' '); i > 0 {
		for j := 2; j < len(numerals); j++ {
			if name[i+1:] == numerals[j] {
				base, n = name[:i], j
				break
			}
		}
	}
	if n+1 >= len(numerals) {
		return name
	}
	next := base + " " + numerals[n+1]
	if len(next) > MaxNameLen { // keep the numeral, shorten the base
		next = strings.TrimSpace(base[:MaxNameLen-len(numerals[n+1])-1]) + " " + numerals[n+1]
	}
	return next
}
