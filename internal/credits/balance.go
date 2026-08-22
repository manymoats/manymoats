package credits

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Balance is the answer to "what's left". Amount is a pointer on purpose:
// nil means unknown, and unknown must never be confused with zero. This is the
// single most important line in this package.
type Balance struct {
	CreditID string
	Source   Method
	Amount   *float64
	Unit     string
	Detail   string
	AsOf     time.Time
	AgeDays  int
	Err      error
}

func (b Balance) Known() bool { return b.Amount != nil }

type BalanceOptions struct {
	Only      []string
	NoNetwork bool
	KeyDir    string
	Client    *http.Client
	Holdings  *Holdings
	Now       time.Time
}

// probeTimeout is the figure the working prototype proved. Eight seconds is
// long enough for a cold TLS handshake and short enough that a dead provider
// does not hold up the other seven credits.
const probeTimeout = 8 * time.Second

type probe func(ctx context.Context, o BalanceOptions, c Credit) (float64, string, error)

var probes = map[string]probe{
	"moonshot": moonshotBalance,
	"fal":      falBalance,
}

// Balances never returns fewer rows than it was asked for. A probe that fails
// demotes that one credit to declared and records why; every other credit is
// still in the slice.
func (cat *Catalog) Balances(ctx context.Context, o BalanceOptions) []Balance {
	now := o.Now
	if now.IsZero() {
		now = time.Now()
	}
	out := make([]Balance, 0, len(cat.Credits))
	for _, c := range cat.Credits {
		if len(o.Only) > 0 && !contains(o.Only, c.ID) {
			continue
		}
		out = append(out, cat.balance(ctx, o, c, now))
	}
	return out
}

func (cat *Catalog) balance(ctx context.Context, o BalanceOptions, c Credit, now time.Time) Balance {
	b := Balance{CreditID: c.ID, Source: Declared, Unit: c.Unit, AsOf: now}
	if o.Holdings != nil {
		if h, ok := o.Holdings.Find(c.ID); ok {
			b.Amount = h.Amount
			b.Detail = h.Detail
			b.AsOf = o.Holdings.AsOfTime()
			b.AgeDays = o.Holdings.AgeDays(now)
		}
	}
	p, wired := probes[c.ID]
	if !wired {
		if b.Amount == nil && b.Err == nil {
			b.Err = errNoProbe
		}
		return b
	}
	if o.NoNetwork {
		b.Err = ErrNoNetwork
		return b
	}
	cctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	v, detail, err := p(cctx, o, c)
	if err != nil {
		b.Err = err
		return b
	}
	b.Source, b.Amount, b.Detail, b.AsOf, b.AgeDays = Live, &v, detail, now, 0
	return b
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

// ── keys ───────────────────────────────────────────────────────────────────
// Read, never written, never logged, never printed, not even truncated.

var (
	errNoKey = errors.New("no key")
	// errNoProbe is the honest state of a credit nobody has proved an endpoint
	// for. An unproven probe ships as no probe at all.
	errNoProbe = errors.New("no balance endpoint has been proven for this one")
	// ErrNoNetwork is exported so a caller can tell "we could not" from
	// "we chose not to".
	ErrNoNetwork = errors.New("--no-network was set, so we did not ask")
)

func key(o BalanceOptions, c Credit) (string, error) {
	for _, name := range c.Source.KeyEnv {
		if v := strings.TrimSpace(os.Getenv(name)); v != "" {
			return v, nil
		}
	}
	dirs := []string{o.KeyDir}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".config", "freecredits"))
		for _, f := range c.Source.KeyFiles {
			dirs = append(dirs, filepath.Dir(filepath.Join(home, ".config", f)))
		}
	}
	var names []string
	names = append(names, "keys.env")
	for _, f := range c.Source.KeyFiles {
		names = append(names, filepath.Base(f))
	}
	for _, d := range dirs {
		if d == "" {
			continue
		}
		for _, n := range names {
			p := filepath.Join(d, n)
			st, err := os.Stat(p)
			if err != nil {
				continue
			}
			if m := st.Mode().Perm(); m&0o077 != 0 {
				return "", fmt.Errorf("key file %s is readable by others (mode %04o) — refused, run chmod 600 on it", p, m)
			}
			b, err := os.ReadFile(p)
			if err != nil {
				continue
			}
			for _, name := range c.Source.KeyEnv {
				re := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(name) + `=(.*)$`)
				if m := re.FindSubmatch(b); m != nil {
					v := strings.TrimSpace(string(m[1]))
					v = strings.Trim(v, `"'`)
					if v != "" {
						return v, nil
					}
				}
			}
		}
	}
	return "", errNoKey
}

// ── probes ─────────────────────────────────────────────────────────────────
// Wired only because a real call returned a real number and the response shape
// was recorded. Half a probe is how a tool starts lying.

func get(ctx context.Context, o BalanceOptions, url, auth string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", auth)
	cl := o.Client
	if cl == nil {
		cl = &http.Client{Timeout: probeTimeout}
	}
	res, err := cl.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, errors.New("timeout")
		}
		return nil, errors.New("could not reach the provider")
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, errors.New("could not read the reply")
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", res.StatusCode)
	}
	return body, nil
}

func moonshotBalance(ctx context.Context, o BalanceOptions, c Credit) (float64, string, error) {
	k, err := key(o, c)
	if err != nil {
		return 0, "", err
	}
	body, err := get(ctx, o, "https://api.moonshot.ai/v1/users/me/balance", "Bearer "+k)
	if err != nil {
		return 0, "", err
	}
	var r struct {
		Data struct {
			Available *float64 `json:"available_balance"`
			Cash      *float64 `json:"cash_balance"`
			Voucher   *float64 `json:"voucher_balance"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &r); err != nil || r.Data.Available == nil {
		return 0, "", errors.New("the reply was not the shape we recorded")
	}
	detail := ""
	if r.Data.Cash != nil && r.Data.Voucher != nil {
		detail = fmt.Sprintf("$%.2f you paid  +  $%.2f of voucher", *r.Data.Cash, *r.Data.Voucher)
	}
	return round2(*r.Data.Available), detail, nil
}

func falBalance(ctx context.Context, o BalanceOptions, c Credit) (float64, string, error) {
	k, err := key(o, c)
	if err != nil {
		return 0, "", err
	}
	body, err := get(ctx, o, "https://rest.alpha.fal.ai/billing/user_balance", "Key "+k)
	if err != nil {
		return 0, "", err
	}
	n, err := strconv.ParseFloat(strings.TrimSpace(string(body)), 64)
	if err != nil {
		return 0, "", errors.New("the reply was not the shape we recorded")
	}
	return round2(n), "", nil
}

func round2(f float64) float64 {
	return float64(int64(f*100+0.5)) / 100
}
