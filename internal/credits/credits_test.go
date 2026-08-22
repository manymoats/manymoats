package credits

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func mk(days int, v Verdict) Credit {
	return mkClock(days, v, Stable)
}

func mkClock(days int, v Verdict, ck Clock) Credit {
	return Credit{Rules: []Rule{{
		Door: "google-ai-studio", Verdict: v, Clock: ck,
		VerifiedOn: time.Now().AddDate(0, 0, -days).Format("2006-01-02"),
	}}}
}

func TestFreshFactIsAsserted(t *testing.T) {
	got, _, _ := mk(10, NotCovered).Coverage("google-ai-studio")
	if got != NotCovered {
		t.Fatalf("fresh fact should assert, got %v", got)
	}
}

func TestStaleFactWarnsButStillAsserts(t *testing.T) {
	c := mk(120, NotCovered)
	if got, _, _ := c.Coverage("google-ai-studio"); got != NotCovered {
		t.Fatalf("120d fact should still assert, got %v", got)
	}
	if !c.Stale("google-ai-studio") {
		t.Fatal("120d fact should be flagged stale")
	}
}

func TestVeryOldFactGoesQuietInsteadOfWrong(t *testing.T) {
	got, age, why := mk(200, NotCovered).Coverage("google-ai-studio")
	if got != Unknown {
		t.Fatalf("a 200d fact must demote to unknown, got %v", got)
	}
	if age < 200 || why == "" {
		t.Fatalf("demotion must explain itself: age=%d why=%q", age, why)
	}
}

func TestUnknownDoorNeverGuesses(t *testing.T) {
	if got, _, _ := mk(1, Covered).Coverage("some-door-nobody-catalogued"); got != Unknown {
		t.Fatalf("an uncatalogued door must be unknown, never inherited: got %v", got)
	}
}

func TestPerDayIsTheDecision(t *testing.T) {
	c := Credit{Balance: 263.63, HasBal: true,
		Expires: time.Now().AddDate(0, 0, 23).Format("2006-01-02")}
	pd, ok := c.PerDay()
	if !ok {
		t.Fatal("should compute")
	}
	if fmt.Sprintf("%.2f", pd) != "11.46" {
		t.Fatalf("got %.2f want 11.46", pd)
	}
}

func TestDaysLeftIsCalendarNotClock(t *testing.T) {
	exp := time.Now().AddDate(0, 0, 23).Format("2006-01-02")
	if d, _ := (Credit{Expires: exp}).DaysLeft(); d != 23 {
		t.Fatalf("23 calendar days out must read 23, got %d — the answer must not depend on the time of day", d)
	}
}

func TestVolatileFactRotsFastEnoughToHaveCaughtKimiK3(t *testing.T) {
	// The real case: verified 2026-08-15, already wrong by 2026-08-21. Six days.
	// The volatile warn window must sit UNDER that gap or this test is theatre.
	if warn, _ := Volatile.windows(); warn >= 6 {
		t.Fatalf("volatile warn is %dd — would not have caught the kimi-k3 miss", warn)
	}
	old := time.Now().AddDate(0, 0, -6).Format("2006-01-02")
	c := Credit{Rules: []Rule{
		{Door: "included-model", Verdict: NotCovered, VerifiedOn: old},
		{Door: "grant-terms", Verdict: Covered, VerifiedOn: old, Clock: Stable},
	}}
	if !c.Stale("included-model") {
		t.Error("a 6d-old plan-inclusion fact must already be flagged stale")
	}
	if c.Stale("grant-terms") {
		t.Error("a 6d-old published-terms fact must not be flagged")
	}
	if v, _, _ := c.Coverage("included-model"); v != NotCovered {
		t.Errorf("stale is a warning, not a demotion, at 6d: got %v", v)
	}
	older := time.Now().AddDate(0, 0, -15).Format("2006-01-02")
	c.Rules[0].VerifiedOn = older
	if v, _, why := c.Coverage("included-model"); v != Unknown {
		t.Errorf("past the demote window a volatile fact must go quiet: got %v (%s)", v, why)
	}
}

// Every rule that claims the long shelf life has to name a published source for it.
func TestStableClockRequiresACitation(t *testing.T) {
	cs, err := Catalog()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cs {
		for _, r := range c.Rules {
			if r.Clock == Stable && r.Source == "" && r.Note == "" {
				t.Errorf("%s/%s claims the stable clock with nothing to back it", c.ID, r.Door)
			}
		}
	}
}

// The binary must not carry one person's credits to everyone who installs it.
func TestShippedFactsAreNobodysInParticular(t *testing.T) {
	cs, err := Providers()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cs {
		if c.Expires != "" {
			t.Errorf("%s ships an expiry date — that is the installer's clock, not ours", c.ID)
		}
		for _, r := range c.Rules {
			// Markers of one person's account rather than a shared fact.
			for _, leak := range []string{"workspace ", "founder", "/Users/", "my ", "$0.00"} {
				if strings.Contains(strings.ToLower(r.Note), leak) {
					t.Errorf("%s/%s note reads as one account's data (%q) — that ships to every install", c.ID, r.Door, leak)
				}
			}
		}
	}
}
