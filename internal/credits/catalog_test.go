package credits

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func load(t *testing.T) *Catalog {
	t.Helper()
	cat, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	return cat
}

func TestTheTwoDoorAnswerIsTheWholeProduct(t *testing.T) {
	cat := load(t)
	paid := cat.Covers("gcp-trial", "vertex-ai", "gemini-3.7-flash")
	billed := cat.Covers("gcp-trial", "google-ai-studio", "gemini-3.7-flash")
	if paid.Verdict != Covered {
		t.Errorf("the trial must pay for Gemini through Vertex, got %v", paid.Verdict)
	}
	if billed.Verdict != NotCovered {
		t.Errorf("the trial must NOT pay for the same model through AI Studio, got %v", billed.Verdict)
	}
	if billed.Why == "" || billed.Source == "" {
		t.Error("the door that bills you has to say why, and where we read it")
	}
}

// The correction that started the two-clock split. If this ever reads
// "not covered" again, something re-imported the stale 2026-08-15 measurement.
func TestKimiK3IsCoveredOnOllamaCloud(t *testing.T) {
	cat := load(t)
	for _, m := range []string{"kimi-k3", "kimi-k3:cloud"} {
		got := cat.Covers("ollama-cloud", "ollama-cloud", m)
		if got.Verdict != Covered {
			t.Errorf("%s on Ollama Cloud must be covered, got %v", m, got.Verdict)
		}
	}
}

// A fact about what a subscription includes must never sit on the long clock.
// That is precisely the mistake the kimi-k3 miss was made of.
func TestPlanInclusionFactsAreOnTheShortClock(t *testing.T) {
	cat := load(t)
	c, ok := cat.Credit("ollama-cloud")
	if !ok {
		t.Fatal("ollama-cloud missing from the catalog")
	}
	for _, r := range c.Rules {
		if r.Clock == Stable {
			t.Errorf("%s/%s claims the stable clock for a plan-inclusion fact", c.ID, r.Door)
		}
	}
}

func TestResolutionOrderPrefersTheLongestPattern(t *testing.T) {
	rules := []Rule{
		{Door: "d", Models: []string{"*"}, Verdict: NotCovered, VerifiedOn: today()},
		{Door: "d", Models: []string{"gemini-*"}, Verdict: Covered, VerifiedOn: today()},
		{Door: "d", Models: []string{"gemini-3.7-flash"}, Verdict: NotCovered, VerifiedOn: today()},
	}
	if _, pat, _ := pick(rules, "d", "gemini-3.7-flash"); pat != "gemini-3.7-flash" {
		t.Errorf("exact id must win, matched %q", pat)
	}
	if _, pat, _ := pick(rules, "d", "gemini-3.9-pro"); pat != "gemini-*" {
		t.Errorf("longest glob must win over the catch-all, matched %q", pat)
	}
	if _, pat, _ := pick(rules, "d", "llama-4"); pat != "*" {
		t.Errorf("catch-all is the last resort, matched %q", pat)
	}
}

func TestSilenceIsUnknownAndNeverInherited(t *testing.T) {
	cat := load(t)
	got := cat.Covers("gcp-trial", "vertex-ai", "some-model-nobody-catalogued")
	if got.Verdict != Unknown {
		t.Fatalf("an uncatalogued model must be unknown, got %v", got.Verdict)
	}
	if got.Why == "" {
		t.Error("unknown has to say why it is unknown")
	}
}

// A ["*"] rule answers a question that names a door. It is not evidence that a
// particular model id lives behind that door, and letting it act like evidence
// buried the two-door answer under a speech-to-text credit.
func TestCatchAllDoesNotAnswerAModelQuestion(t *testing.T) {
	cat := load(t)
	for _, cv := range cat.DoorsFor("assemblyai", "gemini-3.7-flash") {
		t.Errorf("a catch-all made AssemblyAI answer about a Gemini model: %+v", cv)
	}
	if got := cat.Covers("assemblyai", "assemblyai-batch", "anything"); got.Verdict != Covered {
		t.Errorf("naming the door must still get the catch-all answer, got %v", got.Verdict)
	}
}

func TestGlobMatch(t *testing.T) {
	cases := []struct {
		pat, s string
		want   bool
	}{
		{"*", "anything", true},
		{"gemini-*", "gemini-3.7-flash", true},
		{"gemini-*", "gemma-2", false},
		{"*:free", "qwen/qwen3:free", true},
		{"*:free", "qwen/qwen3", false},
		{"kimi-k3", "kimi-k3", true},
		{"kimi-k3", "kimi-k3:cloud", false},
		{"deepseek-v4-*", "deepseek-v4-pro", true},
	}
	for _, c := range cases {
		if got := globMatch(c.pat, c.s); got != c.want {
			t.Errorf("globMatch(%q,%q) = %v, want %v", c.pat, c.s, got, c.want)
		}
	}
}

func TestEveryDoorAndDateResolves(t *testing.T) {
	cat := load(t)
	if len(cat.Credits) == 0 {
		t.Fatal("the built-in catalog is empty")
	}
	for _, c := range cat.Credits {
		if len(c.Sources) == 0 {
			t.Errorf("%s ships with no source at all", c.ID)
		}
		for _, r := range c.Rules {
			if _, ok := cat.Doors[r.Door]; !ok {
				t.Errorf("%s names door %q with no definition", c.ID, r.Door)
			}
			if _, err := time.Parse("2006-01-02", r.VerifiedOn); err != nil {
				t.Errorf("%s/%s has an unparseable verified_on %q", c.ID, r.Door, r.VerifiedOn)
			}
		}
	}
}

// A date in the future would let a fact never age.
func TestNoVerifiedOnIsInTheFuture(t *testing.T) {
	cat := load(t)
	for _, c := range cat.Credits {
		for _, r := range c.Rules {
			if factAge(r.VerifiedOn) < 0 {
				t.Errorf("%s/%s is dated in the future: %s", c.ID, r.Door, r.VerifiedOn)
			}
		}
	}
}

func TestCatalogRefusesAnUnbackedStableClock(t *testing.T) {
	cat := &Catalog{
		Doors:   map[string]Door{"d": {ID: "d"}},
		Credits: []Credit{{ID: "x", Rules: []Rule{{Door: "d", Verdict: Covered, VerifiedOn: today(), Clock: Stable}}}},
	}
	if err := cat.check(); err == nil {
		t.Fatal("a stable-clock rule with no source and no note must be refused")
	}
}

// ── balances ───────────────────────────────────────────────────────────────

func TestAmountIsAPointerSoUnknownIsNeverZero(t *testing.T) {
	cat := load(t)
	bals := cat.Balances(context.Background(), BalanceOptions{NoNetwork: true})
	for _, b := range bals {
		if b.Amount == nil && b.Known() {
			t.Errorf("%s: nil amount reported as known", b.CreditID)
		}
	}
	zero := 0.0
	if !(Balance{Amount: &zero}).Known() {
		t.Error("a real zero balance is a known figure, not an unknown one")
	}
}

// A failed probe demotes that one credit and never takes the run with it.
func TestAFailedProbeDemotesAndEveryOtherCreditStillPrints(t *testing.T) {
	cat := load(t)
	dir := t.TempDir()
	bals := cat.Balances(context.Background(), BalanceOptions{
		KeyDir: dir,
		Client: &http.Client{Transport: deadTransport{}},
	})
	if len(bals) != len(cat.Credits) {
		t.Fatalf("got %d balances for %d credits — a failure ate a row", len(bals), len(cat.Credits))
	}
	var found bool
	for _, b := range bals {
		if b.CreditID != "moonshot" {
			continue
		}
		found = true
		if b.Source != Declared {
			t.Errorf("a failed probe must demote to declared, got %v", b.Source)
		}
		if b.Err == nil {
			t.Error("a demotion has to carry its reason")
		}
		if b.Amount != nil {
			t.Error("a failed probe must never fabricate a figure")
		}
	}
	if !found {
		t.Error("moonshot vanished from the results")
	}
}

type deadTransport struct{}

func (deadTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("network is down")
}

func TestNoNetworkContactsNobody(t *testing.T) {
	cat := load(t)
	bals := cat.Balances(context.Background(), BalanceOptions{
		NoNetwork: true,
		Client:    &http.Client{Transport: forbidden{t}},
	})
	for _, b := range bals {
		if b.Amount != nil {
			t.Errorf("%s produced a figure with the network off", b.CreditID)
		}
	}
}

type forbidden struct{ t *testing.T }

func (f forbidden) RoundTrip(r *http.Request) (*http.Response, error) {
	f.t.Errorf("--no-network still reached out to %s", r.URL.Host)
	return nil, errors.New("blocked")
}

func TestAWorldReadableKeyFileIsRefused(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "keys.env")
	if err := os.WriteFile(p, []byte("MOONSHOT_API_KEY=secret\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := Credit{Source: BalanceSource{KeyEnv: []string{"MOONSHOT_API_KEY"}}}
	_, err := key(BalanceOptions{KeyDir: dir}, c)
	if err == nil {
		t.Fatal("a key file others can read must be refused")
	}
	if got := err.Error(); !strings.Contains(got, "chmod 600") {
		t.Errorf("the refusal has to say what to do about it, got %q", got)
	}
}

func TestAKeyIsNeverPutInAnError(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "keys.env")
	if err := os.WriteFile(p, []byte("MOONSHOT_API_KEY=sk-do-not-leak-me\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	c := Credit{ID: "moonshot", Source: BalanceSource{KeyEnv: []string{"MOONSHOT_API_KEY"}}}
	_, _, err := moonshotBalance(context.Background(),
		BalanceOptions{KeyDir: dir, Client: &http.Client{Transport: deadTransport{}}}, c)
	if err == nil {
		t.Fatal("expected the dead transport to fail")
	}
	if got := err.Error(); got != "could not reach the provider" {
		t.Fatalf("probe errors must not carry the request: %q", got)
	}
}

// ── holdings ───────────────────────────────────────────────────────────────

func TestHoldingsAgeIsCalendarDays(t *testing.T) {
	h := &Holdings{AsOf: time.Now().AddDate(0, 0, -3).Format("2006-01-02")}
	if got := h.AgeDays(time.Now()); got != 3 {
		t.Fatalf("age is %d, want 3 — the answer must not depend on the time of day", got)
	}
}

func TestNoHoldingsFileIsNormalNotAnError(t *testing.T) {
	h, err := LoadHoldings(filepath.Join(t.TempDir(), "nope.json"))
	if err != nil {
		t.Fatalf("a missing holdings file must not be an error: %v", err)
	}
	if h != nil {
		t.Fatal("expected nil holdings")
	}
}

func today() string { return time.Now().Format("2006-01-02") }

// The overlay path runs on every single invocation, so it gets a test.
func TestAnOverlayReplacesAndSaysWhatItReplaced(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "credits"), 0o755); err != nil {
		t.Fatal(err)
	}
	newer := `{"id":"gcp-trial","name":"Google Cloud free trial","coverage":[
	  {"door":"vertex-ai","models":["gemini-*"],"verdict":"not covered","verified_on":"` + today() + `"}]}`
	extra := `{"id":"brand-new","name":"Something Nobody Shipped","sources":["https://example.test"],
	  "coverage":[{"door":"vertex-ai","models":["*"],"verdict":"covered","verified_on":"` + today() + `"}]}`
	for name, body := range map[string]string{"gcp-trial.json": newer, "brand-new.json": extra} {
		if err := os.WriteFile(filepath.Join(dir, "credits", name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cat, err := Load(WithDir(dir))
	if err != nil {
		t.Fatal(err)
	}
	if len(cat.Overlaid) != 1 || cat.Overlaid[0] != "gcp-trial" {
		t.Errorf("the overlay must name what it replaced, got %v", cat.Overlaid)
	}
	if got := cat.Covers("gcp-trial", "vertex-ai", "gemini-3.7-flash"); got.Verdict != NotCovered {
		t.Errorf("the overlay did not win, got %v", got.Verdict)
	}
	if _, ok := cat.Credit("brand-new"); !ok {
		t.Error("an overlay credit with no built-in twin should be added")
	}
}

func TestAMissingOverlayDirectoryIsNotAnError(t *testing.T) {
	if _, err := Load(WithDir(filepath.Join(t.TempDir(), "nope"))); err != nil {
		t.Fatalf("a missing overlay directory must be fine: %v", err)
	}
}

func TestABrokenOverlayFailsLoudlyRatherThanSilently(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "credits"), 0o755); err != nil {
		t.Fatal(err)
	}
	bad := `{"id":"x","coverage":[{"door":"no-such-door","models":["*"],"verdict":"covered","verified_on":"` + today() + `"}]}`
	if err := os.WriteFile(filepath.Join(dir, "credits", "x.json"), []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(WithDir(dir)); err == nil {
		t.Fatal("a verdict naming a door nothing defines must be refused, not answered")
	}
}
