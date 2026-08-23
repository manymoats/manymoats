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

func TestClassifyAliveFalseIsIdle(t *testing.T) {
	// A dead process is idle even when the jsonl is fresh, ends on a
	// question, or has been quiet long enough that an alive session
	// would stall.
	cases := []struct {
		last string
		q    bool
		idle time.Duration
	}{
		{"assistant", false, time.Second},
		{"assistant", true, 20 * time.Minute},
		{"assistant", false, 20 * time.Minute},
		{"user", false, 2 * time.Hour},
	}
	for _, c := range cases {
		if got := Classify(c.last, c.q, true, c.idle, false); got != Idle {
			t.Errorf("alive=false last=%s q=%v idle=%s: got %v, want idle", c.last, c.q, c.idle, got)
		}
	}
}

func TestOnlyActiveIsWorkingAndAsks(t *testing.T) {
	as := []Agent{
		{State: Working, Project: "a"},
		{State: Asks, Project: "b"},
		{State: Stalled, Project: "c"},
		{State: Idle, Project: "d"},
		{State: Done, Project: "e"},
		{State: Resident, Project: "f"},
	}
	got := OnlyActive(as)
	if len(got) != 2 {
		t.Fatalf("only working+asks belong on the default board, got %d", len(got))
	}
	if got[0].State != Working || got[1].State != Asks {
		t.Fatalf("got %v %v", got[0].State, got[1].State)
	}
}

func TestWorkingSidechainStaysOnTheBoard(t *testing.T) {
	as := []Agent{
		{Project: "p", State: Idle},
		{Project: "p", State: Working, Sidechain: true},
		{Project: "p", State: Idle, Sidechain: true},
	}
	got := RollUpSubagents(as)
	if len(got) != 2 {
		t.Fatalf("parent + working child, got %d rows", len(got))
	}
	if got[0].Subagents != 2 {
		t.Fatalf("parent should still count both children, got %d", got[0].Subagents)
	}
	if !got[1].Sidechain || got[1].State != Working {
		t.Fatal("a working subagent must stay visible")
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
