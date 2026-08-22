package conf

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBrokenConfigNeverStopsTheApp(t *testing.T) {
	d := t.TempDir()
	t.Setenv("HOME", d)
	_ = os.MkdirAll(filepath.Join(d, ".orch"), 0o700)
	p, _ := Path()
	_ = os.WriteFile(p, []byte("{ this is not json"), 0o600)
	c := Load()
	if len(c.Machines) == 0 {
		t.Fatal("a broken config must yield defaults — refusing to start over a settings file is worse than a wrong nickname")
	}
}

func TestNicknamesRoundTrip(t *testing.T) {
	d := t.TempDir()
	t.Setenv("HOME", d)
	want := Config{Machines: []Machine{{"big", "local"}, {"little", "little"}}, Names: "brand"}
	if err := Save(want); err != nil {
		t.Fatal(err)
	}
	got := Load()
	if len(got.Machines) != 2 || got.Machines[0].Name != "big" || got.Machines[1].Name != "little" {
		t.Fatalf("nicknames lost: %+v", got.Machines)
	}
	if got.Names != "brand" {
		t.Fatal("preferences lost")
	}
}

func TestFirstRunWritesAStarterFile(t *testing.T) {
	d := t.TempDir()
	t.Setenv("HOME", d)
	c := EnsureExists([]Machine{{"big", "local"}, {"little", "little"}})
	if len(c.Machines) != 2 {
		t.Fatal("discovered machines should seed the starter config")
	}
	p, _ := Path()
	if _, err := os.Stat(p); err != nil {
		t.Fatal("the file people are meant to edit must exist after first run")
	}
}
