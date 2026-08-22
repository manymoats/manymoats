package credits

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Door struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	BelongsTo string `json:"belongs_to,omitempty"`
	Host      string `json:"host,omitempty"`
}

type Catalog struct {
	Credits  []Credit
	Doors    map[string]Door
	Overlaid []string
}

type Option func(*loadOpts)

type loadOpts struct{ dir string }

// WithDir points Load at an overlay directory. The overlay never silently
// wins: every entry it replaces is named in Catalog.Overlaid.
func WithDir(dir string) Option { return func(o *loadOpts) { o.dir = dir } }

func Load(opts ...Option) (*Catalog, error) {
	var o loadOpts
	for _, f := range opts {
		f(&o)
	}
	cat := &Catalog{Doors: map[string]Door{}}

	var doors []Door
	if err := json.Unmarshal(doorsJSON, &doors); err != nil {
		return nil, fmt.Errorf("built-in doors: %w", err)
	}
	for _, d := range doors {
		cat.Doors[d.ID] = d
	}

	names, err := builtinCredits.ReadDir("data/credits")
	if err != nil {
		return nil, err
	}
	for _, n := range names {
		b, err := builtinCredits.ReadFile(filepath.Join("data/credits", n.Name()))
		if err != nil {
			return nil, err
		}
		var c Credit
		if err := json.Unmarshal(b, &c); err != nil {
			return nil, fmt.Errorf("built-in %s: %w", n.Name(), err)
		}
		cat.Credits = append(cat.Credits, c)
	}

	if o.dir != "" {
		if err := cat.overlay(o.dir); err != nil {
			return nil, err
		}
	}
	if err := cat.check(); err != nil {
		return nil, err
	}
	cat.Sort()
	return cat, nil
}

func (cat *Catalog) overlay(dir string) error {
	ents, err := os.ReadDir(filepath.Join(dir, "credits"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, "credits", e.Name()))
		if err != nil {
			return err
		}
		var c Credit
		if err := json.Unmarshal(b, &c); err != nil {
			return fmt.Errorf("%s: %w", e.Name(), err)
		}
		replaced := false
		for i := range cat.Credits {
			if cat.Credits[i].ID == c.ID {
				cat.Credits[i] = c
				cat.Overlaid = append(cat.Overlaid, c.ID)
				replaced = true
				break
			}
		}
		if !replaced {
			cat.Credits = append(cat.Credits, c)
		}
	}
	return nil
}

// check refuses a catalog that would let the tool answer confidently from
// nothing: a verdict with no door, a rule with no date.
func (cat *Catalog) check() error {
	for _, c := range cat.Credits {
		for _, r := range c.Rules {
			if _, ok := cat.Doors[r.Door]; !ok {
				return fmt.Errorf("credit %s names door %q, which no door file defines", c.ID, r.Door)
			}
			if r.VerifiedOn == "" {
				return fmt.Errorf("credit %s has a verdict for %s with no verified_on date", c.ID, r.Door)
			}
			// A long shelf life is earned, never inherited. The rule that went
			// wrong in six days was one nobody had to justify.
			if r.Clock == Stable && r.Source == "" && r.Note == "" {
				return fmt.Errorf("credit %s claims the stable clock for %s with nothing to back it", c.ID, r.Door)
			}
		}
	}
	return nil
}

func (cat *Catalog) Sort() {
	sort.SliceStable(cat.Credits, func(i, j int) bool {
		a, b := cat.Credits[i], cat.Credits[j]
		if a.Dying() != b.Dying() {
			return a.Dying() < b.Dying()
		}
		return a.Name < b.Name
	})
}

func (cat *Catalog) Credit(id string) (Credit, bool) {
	for _, c := range cat.Credits {
		if c.ID == id {
			return c, true
		}
	}
	return Credit{}, false
}

// DoorName is never printed alone. The id is a key; this is the product name a
// reader can search for.
func (cat *Catalog) DoorName(id string) string {
	if d, ok := cat.Doors[id]; ok && d.Name != "" {
		return d.Name
	}
	return id
}

func (cat *Catalog) DoorHost(id string) string { return cat.Doors[id].Host }

type Coverage struct {
	CreditID  string
	DoorID    string
	Model     string
	Verdict   Verdict
	AgeDays   int
	Why       string
	Source    string
	Clock     Clock
	What      string
	Stale     bool
	Withdrawn bool
	Had       Verdict
	Matched   string
}

// Covers answers one question: (credit, door, model). Resolution order is exact
// id, then longest glob, then a catch-all, then nothing — and nothing means
// unknown. There is no "assume covered because the provider is the same", which
// is precisely the assumption that produces the surprise invoice.
func (cat *Catalog) Covers(creditID, doorID, model string) Coverage {
	out := Coverage{CreditID: creditID, DoorID: doorID, Model: model, Verdict: Unknown}
	c, ok := cat.Credit(creditID)
	if !ok {
		out.Why = "we have no record of that credit"
		return out
	}
	r, pat, ok := pick(c.Rules, doorID, model)
	if !ok {
		out.Why = "we have not checked this model at this door"
		return out
	}
	v, age, why := r.Assert()
	out.Verdict, out.AgeDays, out.Why, out.Source, out.Matched = v, age, why, r.Source, pat
	out.What = r.What
	out.Clock = r.Clock
	if r.Withdrawn() {
		out.Withdrawn = true
		out.Had = r.Verdict
	}
	out.Stale = r.StaleRule()
	return out
}

func pick(rules []Rule, door, model string) (Rule, string, bool) {
	var best Rule
	var bestPat string
	found := false
	for _, r := range rules {
		if r.Door != door || len(r.Models) == 0 {
			continue
		}
		for _, p := range r.Models {
			if p == model {
				return r, p, true
			}
			if !globMatch(p, model) {
				continue
			}
			if !found || len(p) > len(bestPat) {
				best, bestPat, found = r, p, true
			}
		}
	}
	return best, bestPat, found
}

// DoorsFor lists every door this credit has a real opinion about for this
// model. A ["*"] catch-all is deliberately NOT a real opinion here: it means
// "everything at this door", which answers a question that names the door, but
// is no evidence at all that some particular model id lives behind it. Letting
// catch-alls answer made "covers gemini-3.7-flash" list a speech-to-text credit
// and bury the two-door answer that the whole tool exists to show.
func (cat *Catalog) DoorsFor(creditID, model string) []Coverage {
	c, ok := cat.Credit(creditID)
	if !ok {
		return nil
	}
	seen := map[string]bool{}
	var out []Coverage
	for _, r := range c.Rules {
		if seen[r.Door] || len(r.Models) == 0 {
			continue
		}
		_, pat, ok := pick(c.Rules, r.Door, model)
		if !ok || pat == "*" {
			continue
		}
		seen[r.Door] = true
		out = append(out, cat.Covers(creditID, r.Door, model))
	}
	return out
}

// Disagrees reports the case worth paying for: one credit, one model, two
// doors, and only one of them free.
func (cat *Catalog) Disagrees(cov []Coverage) bool {
	var yes, no bool
	for _, c := range cov {
		switch c.Verdict {
		case Covered:
			yes = true
		case NotCovered:
			no = true
		}
	}
	return yes && no
}

// globMatch supports '*' and nothing else. A pattern language nobody has to
// learn is worth more here than one that can express everything.
func globMatch(pattern, s string) bool {
	if pattern == "*" {
		return true
	}
	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return pattern == s
	}
	if !strings.HasPrefix(s, parts[0]) {
		return false
	}
	s = s[len(parts[0]):]
	for i := 1; i < len(parts)-1; i++ {
		j := strings.Index(s, parts[i])
		if j < 0 {
			return false
		}
		s = s[j+len(parts[i]):]
	}
	last := parts[len(parts)-1]
	return strings.HasSuffix(s, last) && len(s) >= len(last)
}
