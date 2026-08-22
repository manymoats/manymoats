// Package eyes measures what a surface prints against what is actually there.
//
// It exists because of one night: five confident findings that were fabricated,
// three sets of evidence that hid the very dimension they were meant to show,
// and a render agent that printed "141 characters" over a 148-character draft
// while its own harness had already measured 148. Nobody compared the two.
//
// So the core check is dull and nobody ran it: does the number on the screen
// equal the number in the thing it describes.
package eyes

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
)

// Verdict is deliberately three-valued. "I cannot see this from here" is a
// result, not a failure to produce one — the most useful sentence any judge
// produced in this house was "I cannot verify the minimal view is alive", and
// it found a frozen view four confident passes had missed.
type Verdict string

const (
	Agrees     Verdict = "agrees"
	Disagrees  Verdict = "disagrees"
	Unmeasured Verdict = "not measured"
)

// A Claim is something the surface said, beside what was measured for it.
type Claim struct {
	Said     string
	Measured string
	Verdict  Verdict
	Why      string // populated when unmeasured, or when it disagrees
}

// A Report is one run. Unmeasured is separate from the claims on purpose: it is
// a list of dimensions this run could not see, and it is never a pass.
type Report struct {
	Subject    string
	Claims     []Claim
	Unmeasured []string
	WidestCell int
	WidestByte int
}

var ansi = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// Strip removes styling so text can be measured. Nothing else should strip:
// comparing stripped output hides colour entirely, which is how the most
// animated surface in this house was reported as the only static one.
func Strip(s string) string { return ansi.ReplaceAllString(s, "") }

// Cells is display width. Bytes are not columns — a box-drawing glyph is three
// bytes and one cell, and reporting the byte count as a width already cost this
// house one wrong verdict.
func Cells(s string) int { return lipgloss.Width(s) }

var countClaim = regexp.MustCompile(`(?i)\b(\d[\d,]*)\s+([a-z][a-z ]{2,30}?)\b`)

// Counts finds "N thing" claims and returns HOW MANY, not a row each. With no
// way to count the things themselves, a row per claim is five lines of noise
// saying one thing — and a fragment like "1 not", cut out of "1 not shown",
// reads as a finding when it is a parse artefact.
func Counts(surface string) int {
	seen := map[string]bool{}
	for _, m := range countClaim.FindAllStringSubmatch(Strip(surface), -1) {
		seen[m[1]+"|"+strings.TrimSpace(m[2])] = true
	}
	return len(seen)
}

// Widths measures every line and returns the report's width facts, stating both
// bases so a reader cannot mistake one for the other.
func Widths(surface string, limit int) (cells, bytes int, over []string) {
	for _, l := range strings.Split(Strip(surface), "\n") {
		c, b := Cells(l), len(l)
		if c > cells {
			cells = c
		}
		if b > bytes {
			bytes = b
		}
		if limit > 0 && c > limit {
			over = append(over, l)
		}
	}
	return
}

// Motion compares frames of the SAME state. Every frame must come from a frozen
// subject: captured live, a fresh process re-reads its data and the digits move
// for reasons that have nothing to do with animation — evidence that would have
// confirmed the very claim it was meant to refute.
func Motion(frames []string) (moved [][2]rune, payload [][2]rune) {
	if len(frames) < 2 {
		return nil, nil
	}
	base := []rune(Strip(frames[0]))
	seen := map[[2]rune]bool{}
	for _, f := range frames[1:] {
		cur := []rune(Strip(f))
		for i := range base {
			if i >= len(cur) || base[i] == cur[i] {
				continue
			}
			pair := [2]rune{base[i], cur[i]}
			if seen[pair] {
				continue
			}
			seen[pair] = true
			moved = append(moved, pair)
			if isPayload(base[i]) || isPayload(cur[i]) {
				payload = append(payload, pair)
			}
		}
	}
	return
}

// isPayload: anything a reader reads. Indicator glyphs — block elements, braille,
// the pulse dots — carry no value and may move. A digit may not.
func isPayload(r rune) bool {
	switch {
	case r >= 0x2580 && r <= 0x259F, r >= 0x2800 && r <= 0x28FF:
		return false
	case r == '·' || r == '•':
		return false
	}
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}
