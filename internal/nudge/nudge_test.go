package nudge

import (
	"os"
	"path/filepath"
	"testing"
)

func tmpHome(t *testing.T) {
	t.Helper()
	d := t.TempDir()
	t.Setenv("HOME", d)
	_ = os.MkdirAll(filepath.Join(d, ".orch"), 0o700)
}

func TestDeliveredExactlyOnce(t *testing.T) {
	tmpHome(t)
	if err := Send("claude", "keep cooking"); err != nil {
		t.Fatal(err)
	}
	first, _ := Take("claude")
	if len(first) != 1 || first[0].Text != "keep cooking" {
		t.Fatalf("first take should deliver it, got %+v", first)
	}
	second, _ := Take("claude")
	if len(second) != 0 {
		t.Fatal("delivered twice — a nudge arriving again reads as the founder repeating himself")
	}
}

func TestAddressedNudgesDoNotLeak(t *testing.T) {
	tmpHome(t)
	Send("cursor", "for cursor only")
	if got, _ := Take("claude"); len(got) != 0 {
		t.Fatal("claude took a note addressed to cursor")
	}
	if got, _ := Take("cursor"); len(got) != 1 {
		t.Fatal("cursor did not receive its own note")
	}
}

func TestAnyReachesWhoeverAsksFirst(t *testing.T) {
	tmpHome(t)
	Send("any", "whoever is up")
	if got, _ := Take("claude"); len(got) != 1 {
		t.Fatal("an unaddressed nudge must reach the first asker")
	}
}

func TestPendingCountIsHonest(t *testing.T) {
	tmpHome(t)
	Send("claude", "one")
	Send("claude", "two")
	if Pending("claude") != 2 {
		t.Fatalf("got %d want 2", Pending("claude"))
	}
	Take("claude")
	if Pending("claude") != 0 {
		t.Fatal("pending must clear after delivery")
	}
}
