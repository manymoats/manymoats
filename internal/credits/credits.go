package credits

import (
	"encoding/json"
	"fmt"
	"math"
	"time"
)

type Method string

const (
	Live     Method = "live"
	Derived  Method = "derived"
	Declared Method = "declared"
)

func (m Method) Marker() string {
	switch m {
	case Live:
		return "●"
	case Derived:
		return "◐"
	default:
		return "○"
	}
}

type Verdict string

const (
	Covered    Verdict = "covered"
	NotCovered Verdict = "not covered"
	Unknown    Verdict = "unknown"
)

type Rule struct {
	Door       string   `json:"door"`
	Models     []string `json:"models,omitempty"`
	What       string   `json:"what,omitempty"`
	Verdict    Verdict  `json:"verdict"`
	Note       string   `json:"note,omitempty"`
	VerifiedOn string   `json:"verified_on"`
	Source     string   `json:"source,omitempty"`
	Clock      Clock    `json:"clock,omitempty"`
}

type Expiry struct {
	Kind string `json:"kind"`
	Date string `json:"date,omitempty"`
	Days int    `json:"days,omitempty"`
}

type BalanceSource struct {
	Method    string   `json:"method"`
	Where     string   `json:"where,omitempty"`
	WhereURL  string   `json:"where_url,omitempty"`
	WhyNotAPI string   `json:"why_not_api,omitempty"`
	KeyEnv    []string `json:"key_env,omitempty"`
	KeyFiles  []string `json:"key_files,omitempty"`
}

type Credit struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Colour   string `json:"colour"`
	Expires  string `json:"expires,omitempty"`
	Leftover string `json:"leftover,omitempty"`
	Rules    []Rule `json:"coverage"`

	SchemaVersion int           `json:"schema_version,omitempty"`
	Kind          string        `json:"kind,omitempty"`
	Unit          string        `json:"unit,omitempty"`
	FaceValue     *float64      `json:"face_value,omitempty"`
	HowYouGetIt   string        `json:"how_you_get_it,omitempty"`
	Trap          string        `json:"trap,omitempty"`
	Expiry        Expiry        `json:"expiry,omitempty"`
	Source        BalanceSource `json:"balance,omitempty"`
	VerifiedOn    string        `json:"verified_on,omitempty"`
	Sources       []string      `json:"sources,omitempty"`

	Balance float64 `json:"-"`
	HasBal  bool    `json:"-"`
	Method  Method  `json:"-"`
	AgeDays int     `json:"-"`
	Why     string  `json:"-"`
	Burning string  `json:"-"`
}

// A Clock is how fast this KIND of fact rots. Default is volatile: a provider can
// change what its plan includes overnight, and on 2026-08-21 one did — kimi-k3 went
// 402 to 200 in six days while a 90-day clock still called the old answer fresh.
// Stable is for published terms (grant size, expiry, licence tier) and must be
// declared per rule, so long shelf life is earned rather than inherited.
type Clock string

const (
	Volatile Clock = ""
	Stable   Clock = "stable"
)

func (c Clock) windows() (warn, demote int) {
	if c == Stable {
		return 90, 180
	}
	return 3, 14
}

func (c Credit) DaysLeft() (int, bool) {
	if c.Expires == "" {
		return 0, false
	}
	t, err := time.ParseInLocation("2006-01-02", c.Expires, time.Local)
	if err != nil {
		return 0, false
	}
	n := time.Now()
	today := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.Local)
	return int(math.Round(t.Sub(today).Hours() / 24)), true
}

func (c Credit) PerDay() (float64, bool) {
	d, ok := c.DaysLeft()
	if !ok || d <= 0 || !c.HasBal {
		return 0, false
	}
	return c.Balance / float64(d), true
}

func factAge(verifiedOn string) int {
	t, err := time.Parse("2006-01-02", verifiedOn)
	if err != nil {
		return 1 << 20
	}
	n := time.Now()
	today := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.Local)
	return int(math.Round(today.Sub(t).Hours() / 24))
}

// Assert is the ageing decision, lifted out of Coverage unchanged so that the
// model-aware resolver in catalog.go and the printer both reach their verdict
// by the same two clocks rather than growing a second, quietly different set.
func (r Rule) Assert() (Verdict, int, string) {
	age := factAge(r.VerifiedOn)
	_, demote := r.Clock.windows()
	if age > demote {
		return Unknown, age, "fact is " + fmt.Sprint(age) + "d old — no longer asserted"
	}
	return r.Verdict, age, r.Note
}

func (r Rule) Withdrawn() bool {
	_, demote := r.Clock.windows()
	return factAge(r.VerifiedOn) > demote
}

func (r Rule) StaleRule() bool {
	a := factAge(r.VerifiedOn)
	warn, demote := r.Clock.windows()
	return a > warn && a <= demote
}

func (c Credit) Coverage(door string) (Verdict, int, string) {
	for _, r := range c.Rules {
		if r.Door != door {
			continue
		}
		return r.Assert()
	}
	return Unknown, 0, "no rule for this door"
}

func (c Credit) Stale(door string) bool {
	for _, r := range c.Rules {
		if r.Door == door {
			a := factAge(r.VerifiedOn)
			warn, demote := r.Clock.windows()
			return a > warn && a <= demote
		}
	}
	return false
}

// Decode is orch's Load, renamed only because the plan reserves the name Load
// for the catalog constructor in catalog.go. The body is unchanged.
func Decode(b []byte) ([]Credit, error) {
	var cs []Credit
	return cs, json.Unmarshal(b, &cs)
}

// ExpiryLabel is the "when it dies" cell. A date that parses becomes a
// countdown; everything else says what kind of clock it is, in plain words. It
// never says "no expiry" for something it does not know.
func (c Credit) ExpiryLabel() string {
	if d, ok := c.DaysLeft(); ok {
		switch {
		case d < 0:
			return "already dead"
		case d == 0:
			return "dies today"
		case d == 1:
			return "dies tomorrow"
		default:
			return "dies in " + fmt.Sprint(d) + " days"
		}
	}
	switch c.Expiry.Kind {
	case "days_from_signup":
		if c.Expiry.Days > 0 {
			return "signup + " + fmt.Sprint(c.Expiry.Days) + " days"
		}
	case "days_from_grant":
		if c.Expiry.Days > 0 {
			return "granted + " + fmt.Sprint(c.Expiry.Days) + " days"
		}
	case "weekly_reset":
		return "resets weekly"
	case "monthly_reset":
		return "resets monthly"
	case "per_model":
		return "per-model clocks"
	case "none":
		return "never — metered"
	}
	return string(Unknown)
}

// Dying is the sort key. Everything with a real countdown comes first, soonest
// first, because that is the only ordering this tool asserts on its own — and
// it is division, not opinion.
func (c Credit) Dying() int {
	if d, ok := c.DaysLeft(); ok {
		return d
	}
	return 1 << 20
}
