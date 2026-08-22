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
	Door       string  `json:"door"`
	Verdict    Verdict `json:"verdict"`
	Note       string  `json:"note,omitempty"`
	VerifiedOn string  `json:"verified_on"`
	Source     string  `json:"source,omitempty"`
	Clock      Clock   `json:"clock,omitempty"`
}

type Credit struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Colour   string `json:"colour"`
	Expires  string `json:"expires,omitempty"`
	Leftover string `json:"leftover,omitempty"`
	Rules    []Rule `json:"coverage"`

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

func (c Credit) Coverage(door string) (Verdict, int, string) {
	for _, r := range c.Rules {
		if r.Door != door {
			continue
		}
		age := factAge(r.VerifiedOn)
		_, demote := r.Clock.windows()
		if age > demote {
			return Unknown, age, "fact is " + fmt.Sprint(age) + "d old — no longer asserted"
		}
		return r.Verdict, age, r.Note
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

func Load(b []byte) ([]Credit, error) {
	var cs []Credit
	return cs, json.Unmarshal(b, &cs)
}
