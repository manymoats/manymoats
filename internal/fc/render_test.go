package fc

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/manymoats/manymoats/internal/credits"
)

func testApp(t *testing.T, plain bool, h *credits.Holdings) app {
	t.Helper()
	cat, err := credits.Load()
	if err != nil {
		t.Fatal(err)
	}
	cat.Apply(h)
	a := app{
		cat:  cat,
		bals: map[string]credits.Balance{},
		held: map[string]bool{},
		m:    marks{plain: plain},
		now:  time.Now(),
	}
	for _, b := range cat.Balances(t.Context(), credits.BalanceOptions{NoNetwork: true, Holdings: h}) {
		a.bals[b.CreditID] = b
	}
	if h != nil {
		a.holdings = true
		for _, x := range h.You {
			a.held[x.Credit] = true
		}
	}
	return a
}

func fixtureHoldings(t *testing.T) *credits.Holdings {
	t.Helper()
	amt := func(v float64) *float64 { return &v }
	return &credits.Holdings{
		AsOf: time.Now().AddDate(0, 0, -1).Format("2006-01-02"),
		You: []credits.Holding{
			{Credit: "gcp-trial", Amount: amt(263.63),
				Expires: time.Now().AddDate(0, 0, 22).Format("2006-01-02"), SpentBy: "veo-night3"},
			{Credit: "vertex-promo", Expires: time.Now().AddDate(0, 0, 22).Format("2006-01-02")},
			{Credit: "gcp-genai", Amount: amt(1000),
				Expires: time.Now().AddDate(0, 0, 298).Format("2006-01-02")},
			{Credit: "ollama-cloud", SpentBy: "ask-ollama-cloud"},
			{Credit: "alibaba-model-studio"},
		},
	}
}

// surfaces is every screen this binary can draw. A width test that walks past
// the last real surface is how a new one stops being able to shear quietly.
func surfaces(t *testing.T, a app) map[string]string {
	t.Helper()
	out := map[string]string{
		"../credits":           a.creditsView(),
		"covers":               a.coversView("gemini-3.7-flash"),
		"covers-unknown-model": a.coversView("nothing-we-have-ever-heard-of"),
		"covers-kimi":          a.coversView("kimi-k3"),
		"help":                 helpText(a.m),
		"greet":                greetBody,
	}
	for _, id := range []string{"gcp-trial", "ollama-cloud", "cerebras", "assemblyai"} {
		s, ok := a.showView(id)
		if !ok {
			t.Fatalf("show %s: credit missing", id)
		}
		out["show-"+id] = s
	}
	return out
}

func TestNoSurfaceEverExceedsEightyColumns(t *testing.T) {
	for _, plain := range []bool{false, true} {
		a := testApp(t, plain, fixtureHoldings(t))
		for name, s := range surfaces(t, a) {
			for i, line := range strings.Split(s, "\n") {
				if got := lipgloss.Width(line); got > frameW {
					t.Errorf("plain=%v %s line %d is %d columns: %q", plain, name, i+1, got, line)
				}
			}
		}
	}
}

// The frame is the thing a reader notices first when it drifts.
func TestEveryFrameLineIsExactlyEighty(t *testing.T) {
	for _, plain := range []bool{false, true} {
		a := testApp(t, plain, fixtureHoldings(t))
		n := 0
		for _, line := range strings.Split(a.creditsView(), "\n") {
			if line == "" {
				continue
			}
			n++
			if got := lipgloss.Width(line); got != frameW {
				t.Fatalf("plain=%v frame line %d is %d columns, want %d: %q", plain, n, got, frameW, line)
			}
		}
		if n < 10 {
			t.Fatalf("plain=%v only %d frame lines — the view did not render", plain, n)
		}
	}
}

// The columns, at the exact cells the approved render was measured at.
func TestCreditRowsHoldTheMeasuredColumns(t *testing.T) {
	a := testApp(t, false, fixtureHoldings(t))
	rows := 0
	for _, line := range strings.Split(a.creditsView(), "\n") {
		r := []rune(line)
		if len(r) != frameW || !strings.ContainsRune("●◐○", r[2]) {
			continue
		}
		rows++
		if r[0] != '│' || r[79] != '│' {
			t.Errorf("row lost its walls: %q", line)
		}
		if r[1] != ' ' || r[3] != ' ' {
			t.Errorf("marker gutter drifted: %q", line)
		}
		if seg := string(r[32:43]); seg != strings.TrimRight(seg, " ") {
			t.Errorf("what's left is not right-aligned to column 42: %q", seg)
		}
		if string(r[43:45]) != "  " {
			t.Errorf("the gap before when-it-dies drifted: %q", string(r[43:45]))
		}
	}
	if rows == 0 {
		t.Fatal("no credit rows rendered")
	}
}

// It spends a character to say a character is missing.
func TestNoEllipsisAnywhere(t *testing.T) {
	for _, plain := range []bool{false, true} {
		a := testApp(t, plain, fixtureHoldings(t))
		for name, s := range surfaces(t, a) {
			if strings.ContainsAny(s, "…") {
				t.Errorf("%s (plain=%v) contains an ellipsis", name, plain)
			}
		}
	}
	if got := clip("one two three four five", 9); strings.Contains(got, "…") {
		t.Errorf("clip produced an ellipsis: %q", got)
	}
	if got := clip("Google Cloud · trial", 12); strings.TrimSpace(got) != "Google Cloud" {
		t.Errorf("clip should cut at the separator, got %q", got)
	}
}

// "0 waiting" is noise wearing the costume of information.
func TestNeverPrintsAZero(t *testing.T) {
	a := testApp(t, false, fixtureHoldings(t))
	bad := []string{"0 days old", "0 days ago", "$0.00 a day", "dies in 0 days", "0 day old"}
	for name, s := range surfaces(t, a) {
		for _, b := range bad {
			if strings.Contains(s, b) {
				t.Errorf("%s printed %q", name, b)
			}
		}
	}
}

func TestDeclaredAlwaysCarriesItsAge(t *testing.T) {
	a := testApp(t, false, fixtureHoldings(t))
	for _, line := range strings.Split(a.creditsView(), "\n") {
		r := []rune(line)
		if len(r) != frameW || r[2] != '○' {
			continue
		}
		age := strings.TrimSpace(string(r[63:78]))
		if age == "" {
			t.Errorf("a declared row printed no age at all: %q", line)
		}
		left := strings.TrimSpace(string(r[32:43]))
		if left != "unknown" && age == "never checked" {
			t.Errorf("a figure was printed with no age behind it: %q", line)
		}
	}
}

// Rule one, enforced in the printer as well as the library.
func TestAFigureWithNoDateIsNotShownAsAFigure(t *testing.T) {
	amt := 42.0
	c := credits.Credit{ID: "x", Unit: "usd"}
	undated := credits.Balance{Source: credits.Declared, Amount: &amt, AgeDays: -1}
	if got := leftLabel(c, undated); got != "unknown" {
		t.Fatalf("an undated declared figure must read unknown, got %q", got)
	}
	dated := credits.Balance{Source: credits.Declared, Amount: &amt, AgeDays: 2}
	if got := leftLabel(c, dated); got != "$42.00" {
		t.Fatalf("a dated declared figure should print, got %q", got)
	}
	live := credits.Balance{Source: credits.Live, Amount: &amt}
	if got := ageLabel(live); got != "asked just now" {
		t.Fatalf("live has no age, got %q", got)
	}
}

func TestMoneyCarriesItsUnit(t *testing.T) {
	cases := map[float64]string{263.63: "$263.63", 1000: "$1,000.00", 1234567.5: "$1,234,567.50", 0.5: "$0.50"}
	for in, want := range cases {
		if got := money(in); got != want {
			t.Errorf("money(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestPlainModeSwapsGlyphsAndTheFrameTogether(t *testing.T) {
	a := testApp(t, true, fixtureHoldings(t))
	s := a.creditsView()
	// Box drawing is East-Asian Ambiguous exactly like the markers are, so a
	// fallback that swaps only the markers still shears the frame.
	for _, g := range []string{"●", "◐", "○", "│", "─", "┌", "└"} {
		if strings.Contains(s, g) {
			t.Errorf("plain mode still emitted %q", g)
		}
	}
	for _, wordy := range []string{"live", "old", "+---", "|"} {
		if !strings.Contains(s, wordy) {
			t.Errorf("plain mode did not emit %q", wordy)
		}
	}
}

func TestHelpCarriesTheQuietLineAndIsNeverGated(t *testing.T) {
	h := helpText(marks{})
	for _, want := range []string{
		"no telemetry. we collect nothing.",
		"made by manymoats",
		"A door is the web address you send the request to",
		"not built yet",
	} {
		if !strings.Contains(h, want) {
			t.Errorf("--help is missing %q", want)
		}
	}
	if code := run([]string{"--help"}); code != 0 {
		t.Errorf("--help exited %d", code)
	}
	if code := run([]string{"--version"}); code != 0 {
		t.Errorf("--version exited %d", code)
	}
}

// A normal run carries no mark at all.
func TestANormalRunCarriesNoAttribution(t *testing.T) {
	a := testApp(t, false, fixtureHoldings(t))
	s := a.creditsView()
	for _, mark := range []string{"manymoats.com", "made by", "telemetry"} {
		if strings.Contains(s, mark) {
			t.Errorf("a normal run printed %q", mark)
		}
	}
}

func TestTheTwoDoorCardsAreDrawnAndBothSidesAreEqualWidth(t *testing.T) {
	a := testApp(t, false, fixtureHoldings(t))
	s := a.coversView("gemini-3.7-flash")
	if !strings.Contains(s, "ON YOUR CREDIT") || !strings.Contains(s, "NOT ON IT") {
		t.Fatal("the two-door answer was not drawn")
	}
	if !strings.Contains(s, "same model  ·  two doors  ·  only one is on your credit") {
		t.Error("the punch line is missing")
	}
	if !strings.Contains(s, "aiplatform.googleapis.com") || !strings.Contains(s, "generativelanguage.googleapis.com") {
		t.Error("a door was printed without its real web address")
	}
	// The two-door case must come before anything else, because it is the one
	// that costs money.
	first := strings.Index(s, "paid by")
	if first < 0 || !strings.HasPrefix(strings.TrimSpace(s[first:]), "paid by  Google Cloud free trial") {
		t.Error("the credit that answers two ways is not the first answer on the screen")
	}
}

func TestUnknownModelSaysUnknownAndHowToFindOut(t *testing.T) {
	a := testApp(t, false, fixtureHoldings(t))
	s := a.coversView("model-nobody-has-ever-catalogued")
	if !strings.Contains(s, "unknown") {
		t.Error("an unmatched question must print the word unknown")
	}
	if !strings.Contains(s, "freecredits check") {
		t.Error("unknown has to say how to find out")
	}
}

// The word "pots" and the rest of the house vocabulary never reach a user.
func TestNoInventedVocabularyReachesTheScreen(t *testing.T) {
	banned := []string{"pot", "burn lane", "seat", "SKU", "entitlement", "quota grant", "drift", "stale"}
	a := testApp(t, false, fixtureHoldings(t))
	all := helpText(a.m)
	for _, s := range surfaces(t, a) {
		all += s
	}
	lower := strings.ToLower(all)
	for _, b := range banned {
		if strings.Contains(lower, strings.ToLower(b)) {
			t.Errorf("house vocabulary reached the screen: %q", b)
		}
	}
}

// The library returns values and errors. Nothing in it writes to a stream or
// reads one, so importing it can never make somebody's program print or prompt.
func TestTheLibraryNeverPrintsAndNeverReadsStdin(t *testing.T) {
	dir := "../credits"
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	banned := regexp.MustCompile(`\bfmt\.Print|\bfmt\.Fprint|\bos\.Std(out|err|in)\b|\bprintln\(|\blog\.[A-Z]`)
	for _, e := range ents {
		if !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if m := banned.FindString(string(b)); m != "" {
			t.Errorf("credits/%s uses %s — the library must not touch a stream", e.Name(), m)
		}
	}
}

// A label that outgrows its column gets silently cut, and a cut label reads as
// garbage rather than as missing. Every label has to fit the cell it lives in.
func TestEveryLabelFitsItsColumn(t *testing.T) {
	amt := 12.5
	for _, b := range []credits.Balance{
		{Source: credits.Live, Amount: &amt},
		{Source: credits.Declared},
		{Source: credits.Declared, Amount: &amt, AgeDays: -1},
		{Source: credits.Declared, Amount: &amt, AgeDays: 0},
		{Source: credits.Declared, Amount: &amt, AgeDays: 1},
		{Source: credits.Declared, Amount: &amt, AgeDays: 365},
	} {
		if got := ageLabel(b); lipgloss.Width(got) > ageW {
			t.Errorf("age label %q is %d cells, column is %d", got, lipgloss.Width(got), ageW)
		}
	}
	cat, err := credits.Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cat.Credits {
		if got := c.ExpiryLabel(); lipgloss.Width(got) > diesW {
			t.Errorf("%s expiry label %q is %d cells, column is %d", c.ID, got, lipgloss.Width(got), diesW)
		}
	}
	for _, d := range cat.Doors {
		if lipgloss.Width(d.Name) > frameW-doorCol {
			t.Logf("door %q wraps under its column (%d cells)", d.Name, lipgloss.Width(d.Name))
		}
	}
}

// The tool can know what a credit covers. It cannot know what a call will cost
// you — that depends on a billing account it has never seen. Any wording that
// asserts a charge is a claim this product cannot defend.
func TestNoSurfaceClaimsAChargeItCannotKnow(t *testing.T) {
	banned := []string{"charges your card", "bills you", "will be charged", "you pay", "costs you $"}
	for _, plain := range []bool{false, true} {
		a := testApp(t, plain, fixtureHoldings(t))
		for name, s := range surfaces(t, a) {
			low := strings.ToLower(s)
			for _, b := range banned {
				if strings.Contains(low, b) {
					t.Errorf("%s claims %q — the billing account is not visible to us", name, b)
				}
			}
		}
	}
}

// A column that overflows is invisible to a wording check and obvious to a
// reader: the verdict ran into the door column with no gap the moment the
// wording grew from nine characters to eighteen, and grepping for the phrase
// could not find it because there was no space left to grep for. This checks
// the geometry, so any future wording change is caught by construction.
func TestTableColumnsHoldNoMatterHowLongTheCellIs(t *testing.T) {
	cases := [][3]string{
		{"pays for", "Gemini, Veo, Imagen", "Vertex AI"},
		{"not covered", "Gemini", "Gemini API (AI Studio)"},
		{"a verdict far longer than its column", "what", "door"},
		{"ok", "a what cell that is very much longer than the column it lives in", "door"},
	}
	for _, c := range cases {
		line := []rune(tableRow(c[0], c[1], c[2])[0])
		for _, at := range []int{2 + verdictW, doorCol} {
			if at >= len(line) {
				continue
			}
			if line[at-1] != ' ' {
				t.Errorf("cell %q ran to the column boundary at %d with no gap: %q",
					c[0], at, strings.TrimRight(string(line), " "))
			}
		}
	}
}

// The words may change; they must stay inside the column when they do.
func TestEveryVerdictWordFitsItsColumn(t *testing.T) {
	for _, v := range []credits.Verdict{credits.Covered, credits.NotCovered, credits.Unknown} {
		if w := lipgloss.Width(verdictWord(v)); w >= verdictW {
			t.Errorf("verdict word %q is %d wide, column is %d", verdictWord(v), w, verdictW)
		}
	}
}
