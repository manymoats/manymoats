package credits

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"time"
)

// Holdings is what you wrote down yourself. It is never shipped in the binary,
// never uploaded, and never read by anything but you. Amount is a pointer so a
// credit you hold but cannot check reads unknown instead of zero.
type Holdings struct {
	AsOf string    `json:"as_of"`
	You  []Holding `json:"you_hold"`
	path string    `json:"-"`
	mod  time.Time `json:"-"`
}

type Holding struct {
	Credit  string   `json:"credit"`
	Amount  *float64 `json:"amount,omitempty"`
	Detail  string   `json:"detail,omitempty"`
	Expires string   `json:"expires,omitempty"`
	SpentBy string   `json:"being_spent_by,omitempty"`
}

func DefaultHoldingsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".manymoats", "credits", "holdings.json")
}

// LoadHoldings returns nil with no error when there is no file. Not having one
// is normal, not a failure.
func LoadHoldings(path string) (*Holdings, error) {
	if path == "" {
		return nil, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var h Holdings
	if err := json.Unmarshal(b, &h); err != nil {
		return nil, err
	}
	h.path = path
	if st, err := os.Stat(path); err == nil {
		h.mod = st.ModTime()
	}
	return &h, nil
}

func (h *Holdings) Find(creditID string) (Holding, bool) {
	if h == nil {
		return Holding{}, false
	}
	for _, x := range h.You {
		if x.Credit == creditID {
			return x, true
		}
	}
	return Holding{}, false
}

func (h *Holdings) AsOfTime() time.Time {
	if h == nil {
		return time.Time{}
	}
	if t, err := time.ParseInLocation("2006-01-02", h.AsOf, time.Local); err == nil {
		return t
	}
	return h.mod
}

// AgeDays is why declared exists. A declared figure with no age is a bug, and
// the tests say so.
func (h *Holdings) AgeDays(now time.Time) int {
	t := h.AsOfTime()
	if t.IsZero() {
		return -1
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
	return int(math.Round(today.Sub(d).Hours() / 24))
}

// Apply folds what you wrote down into the catalog: your expiry date, your
// figure, and the job you told us is spending it. It changes no verdict.
func (cat *Catalog) Apply(h *Holdings) {
	if h == nil {
		return
	}
	for i := range cat.Credits {
		x, ok := h.Find(cat.Credits[i].ID)
		if !ok {
			continue
		}
		if x.Expires != "" {
			cat.Credits[i].Expires = x.Expires
		}
		if x.Amount != nil {
			cat.Credits[i].Balance = *x.Amount
			cat.Credits[i].HasBal = true
		}
		cat.Credits[i].Burning = x.SpentBy
	}
	cat.Sort()
}
