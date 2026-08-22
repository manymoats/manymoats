package credits

import (
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"
)

//go:embed data/providers.json
var providersJSON []byte

// A Holding is the part of a credit that belongs to one person: when THEIR
// grant runs out, which workspace it sits in. It never ships in the binary —
// the embedded file carries only facts that are true for everybody.
type Holding struct {
	ID      string `json:"id"`
	Expires string `json:"expires,omitempty"`
	Note    string `json:"note,omitempty"`
}

func holdingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".orch", "credits.json"), nil
}

// Holdings returns what this user actually has. A missing file is the normal
// state for a fresh install, not an error.
func Holdings() []Holding {
	p, err := holdingsPath()
	if err != nil {
		return nil
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	var hs []Holding
	if json.Unmarshal(b, &hs) != nil {
		return nil
	}
	return hs
}

// Providers is the shared knowledge: who offers what, and which of it is real.
func Providers() ([]Credit, error) { return Load(providersJSON) }

// Catalog is Providers with the caller's own expiry dates layered on. Without a
// holdings file it still returns every provider, just with no personal clock —
// so a fresh install shows what exists in the world and claims nothing about
// what you hold.
func Catalog() ([]Credit, error) {
	cs, err := Providers()
	if err != nil {
		return nil, err
	}
	held := map[string]Holding{}
	for _, h := range Holdings() {
		held[h.ID] = h
	}
	for i := range cs {
		if h, ok := held[cs[i].ID]; ok {
			cs[i].Expires = h.Expires
			cs[i].Why = h.Note
		}
	}
	return cs, nil
}
