package agent

import (
	"testing"
	"time"
)

func TestStalledIsReachable(t *testing.T) {
	got := Classify("assistant", false, false, 20*time.Minute, true)
	if got != Stalled {
		t.Fatalf("stalled unreachable: got %v, want stalled", got)
	}
}

func TestEvaluationOrder(t *testing.T) {
	cases := []struct {
		name     string
		lastTurn string
		question bool
		prompt   bool
		idle     time.Duration
		alive    bool
		want     State
	}{
		{"asks beats working", "assistant", true, false, 50 * time.Second, true, Asks},
		{"pending prompt asks", "assistant", false, true, 60 * time.Second, true, Asks},
		{"fresh mtime is working", "assistant", false, false, 5 * time.Second, true, Working},
		{"finished is done not asks", "assistant", false, false, 2 * time.Minute, true, Done},
		{"long silence stalls", "assistant", false, false, 20 * time.Minute, true, Stalled},
		{"user turn is idle", "user", false, false, 5 * time.Minute, true, Idle},
		{"dead process idle", "assistant", true, true, time.Second, false, Idle},
	}
	for _, c := range cases {
		if got := Classify(c.lastTurn, c.question, c.prompt, c.idle, c.alive); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

func TestCollidingAgentsGetDistinguished(t *testing.T) {
	as := []Agent{
		{ID: "aaaa1111", Model: "opus", Project: "kpf"},
		{ID: "bbbb2222", Model: "opus", Project: "kpf"},
		{ID: "cccc3333", Model: "opus", Project: "other"},
	}
	Disambiguate(as)
	if as[0].Tag == "" || as[1].Tag == "" {
		t.Fatal("two agents with the same model AND project must be distinguishable")
	}
	if as[0].Tag == as[1].Tag {
		t.Fatal("their tags must differ or the board still lies")
	}
	if as[2].Tag != "" {
		t.Fatal("a unique agent must not be cluttered with a tag it does not need")
	}
}
