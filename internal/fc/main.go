package fc

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/manymoats/manymoats/internal/credits"
)

// invocation is how this tool is actually typed. It lives in one place so the
// help, the errors and the examples can never disagree about its own name.
const invocation = "manymoats credits"

const (
	version     = "0.1.0"
	catalogDate = "2026-08-21"
)

type opts struct {
	plain     bool
	jsonOut   bool
	yes       bool
	noNetwork bool
	snapshot  bool
	holdings  string
	cmd       string
	arg       string
}

func parse(args []string) (opts, error) {
	o := opts{cmd: "credits"}
	var rest []string
	for i := 0; i < len(args); i++ {
		switch a := args[i]; a {
		case "--plain":
			o.plain = true
		case "--json":
			o.jsonOut = true
		case "--yes", "-y":
			o.yes = true
		case "--no-network":
			o.noNetwork = true
		case "--snapshot", "-s":
			o.snapshot = true
		case "--holdings":
			if i+1 >= len(args) {
				return o, fmt.Errorf("--holdings needs a file path")
			}
			i++
			o.holdings = args[i]
		case "--help", "-h", "help":
			o.cmd = "help"
		case "--version", "-V":
			o.cmd = "version"
		default:
			if strings.HasPrefix(a, "-") {
				return o, fmt.Errorf("unknown option %q — run "+invocation+" --help", a)
			}
			rest = append(rest, a)
		}
	}
	if o.cmd == "help" || o.cmd == "version" {
		return o, nil
	}
	if len(rest) > 0 {
		o.cmd = rest[0]
		if len(rest) > 1 {
			o.arg = rest[1]
		}
	}
	return o, nil
}

// Main is the credits board. It returns an exit code so the manymoats
// dispatcher owns process teardown.
func Main() int { return run(os.Args[1:]) }

func run(args []string) int {
	o, err := parse(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, invocation+":", err)
		return 2
	}

	// --help and --version are never gated, never touch the network, and never
	// load a catalog they might fail on.
	switch o.cmd {
	case "help":
		fmt.Print(helpText(marks{plain: o.plain}))
		return 0
	case "version":
		fmt.Printf(invocation+" %s\ncatalog %s, dated %s\n\n  %s\n", version, "1", catalogDate, makerLine)
		return 0
	}

	a, err := build(o)
	if err != nil {
		fmt.Fprintln(os.Stderr, invocation+":", err)
		return 1
	}

	// The greeting sits above the dispatch on purpose. Hooking it into a view
	// is how it comes back to life in the next view somebody adds.
	greet(o, os.Stdin, os.Stdout)

	switch o.cmd {
	case "credits", "list":
		if o.cmd == "list" {
			a.holdings = false
		}
		if o.jsonOut {
			return emitJSON(a)
		}
		fmt.Print(a.creditsView())
	case "covers":
		if o.arg == "" {
			fmt.Fprintln(os.Stderr, invocation+": covers needs a model name, like: "+invocation+" covers gemini-3.7-flash")
			return 2
		}
		fmt.Print(a.coversView(o.arg))
	case "show":
		if o.arg == "" {
			fmt.Fprintln(os.Stderr, invocation+": show needs a credit id, like: "+invocation+" show gcp-trial")
			return 2
		}
		out, ok := a.showView(o.arg)
		if !ok {
			fmt.Fprintf(os.Stderr, invocation+": no credit called %q. "+invocation+" list shows every one we know.\n", o.arg)
			return 2
		}
		fmt.Print(out)
	default:
		fmt.Fprintf(os.Stderr, invocation+": %q is not built yet. "+invocation+" --help lists what is.\n", o.cmd)
		return 2
	}
	return 0
}

func build(o opts) (app, error) {
	cat, err := credits.Load(credits.WithDir(overlayDir()))
	if err != nil {
		return app{}, err
	}

	path := o.holdings
	if path == "" {
		path = credits.DefaultHoldingsPath()
	}
	h, err := credits.LoadHoldings(path)
	if err != nil {
		return app{}, fmt.Errorf("reading %s: %w", path, err)
	}
	cat.Apply(h)

	now := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	bals := cat.Balances(ctx, credits.BalanceOptions{
		NoNetwork: o.noNetwork,
		Holdings:  h,
		Now:       now,
	})

	a := app{
		cat:  cat,
		bals: map[string]credits.Balance{},
		held: map[string]bool{},
		m:    marks{plain: wordMarkers(o)},
		now:  now,
	}
	for _, b := range bals {
		a.bals[b.CreditID] = b
	}
	if h != nil {
		a.holdings = true
		for _, x := range h.You {
			a.held[x.Credit] = true
		}
	}
	return a, nil
}

// wordMarkers decides whether ● ◐ ○ can be trusted to take one cell. They are
// East-Asian Ambiguous, so a CJK locale or a terminal that says two cells gets
// the words instead. --snapshot renders the terminal's own frame on purpose —
// it is how this binary's alignment gets checked without a terminal.
func wordMarkers(o opts) bool {
	if o.plain {
		return true
	}
	if ambiguousWide() {
		return true
	}
	if o.snapshot {
		return false
	}
	return !isTTY(os.Stdout)
}

func overlayDir() string {
	if d := os.Getenv("FREECREDITS_CATALOG"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home + "/.manymoats/credits/catalog"
}

func emitJSON(a app) int {
	type ruleOut struct {
		Door       string   `json:"door"`
		DoorName   string   `json:"door_name"`
		Models     []string `json:"models,omitempty"`
		Verdict    string   `json:"verdict"`
		AgeDays    int      `json:"age_days"`
		Clock      string   `json:"clock"`
		Withdrawn  bool     `json:"withdrawn,omitempty"`
		Stale      bool     `json:"stale,omitempty"`
		Source     string   `json:"source,omitempty"`
		VerifiedOn string   `json:"verified_on"`
	}
	type out struct {
		ID       string    `json:"id"`
		Name     string    `json:"name"`
		Source   string    `json:"source"`
		Amount   *float64  `json:"amount"`
		Unit     string    `json:"unit,omitempty"`
		AgeDays  *int      `json:"age_days"`
		Expires  string    `json:"expires,omitempty"`
		DiesIn   *int      `json:"dies_in_days"`
		PerDay   *float64  `json:"per_day_to_use_it"`
		Why      string    `json:"why_unknown,omitempty"`
		Coverage []ruleOut `json:"coverage"`
	}
	var docs []out
	for _, c := range a.shown() {
		b := a.balance(c.ID)
		o := out{ID: c.ID, Name: c.Name, Source: string(b.Source), Amount: b.Amount, Unit: c.Unit, Expires: c.Expires}
		if b.Amount != nil {
			c.Balance, c.HasBal = *b.Amount, true
			age := b.AgeDays
			o.AgeDays = &age
		}
		if d, ok := c.DaysLeft(); ok {
			o.DiesIn = &d
		}
		if pd, ok := c.PerDay(); ok {
			o.PerDay = &pd
		}
		if b.Err != nil {
			o.Why = b.Err.Error()
		}
		for _, r := range c.Rules {
			v, age, _ := r.Assert()
			clock := "volatile"
			if r.Clock == credits.Stable {
				clock = "stable"
			}
			o.Coverage = append(o.Coverage, ruleOut{
				Door: r.Door, DoorName: a.cat.DoorName(r.Door), Models: r.Models,
				Verdict: string(v), AgeDays: age, Clock: clock,
				Withdrawn: r.Withdrawn(), Stale: r.StaleRule(),
				Source: r.Source, VerifiedOn: r.VerifiedOn,
			})
		}
		docs = append(docs, o)
	}
	e := json.NewEncoder(os.Stdout)
	e.SetIndent("", "  ")
	if err := e.Encode(docs); err != nil {
		fmt.Fprintln(os.Stderr, invocation+":", err)
		return 1
	}
	return 0
}
