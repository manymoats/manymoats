// Package first is the first run of the main binary: the house splash, one
// question about updates, and `manymoats update`. The word is manymoats.
package first

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Path is ~/.config/manymoats/seen (or $XDG_CONFIG_HOME/manymoats/seen).
// Delete it to see the first-run splash and the updates question again.
func Path() string {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "manymoats", "seen")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "manymoats", "seen")
}

// Has reports whether this machine already recorded kind ("main" or "apps").
func Has(kind string) bool {
	b, err := os.ReadFile(Path())
	if err != nil {
		return false
	}
	for _, ln := range strings.Split(string(b), "\n") {
		if strings.TrimSpace(ln) == kind {
			return true
		}
	}
	return false
}

// recordable answers "can we remember this" before doing it, so a read-only
// home skips the question instead of repeating it forever.
func recordable(path string) bool {
	if path == "" {
		return false
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return false
	}
	f, err := os.OpenFile(path+".probe", os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return false
	}
	f.Close()
	os.Remove(path + ".probe")
	return true
}

// Mark writes kind into the seen file. One file, two notes: "main" is the
// front door, "apps" is a smaller hello. Deleting the file replays both.
func Mark(kind string) {
	p := Path()
	if !recordable(p) {
		return
	}
	kinds := map[string]bool{}
	if b, err := os.ReadFile(p); err == nil {
		for _, ln := range strings.Split(string(b), "\n") {
			switch strings.TrimSpace(ln) {
			case "main", "apps":
				kinds[ln] = true
			}
		}
	}
	kinds[kind] = true
	var b strings.Builder
	b.WriteString(time.Now().UTC().Format(time.RFC3339) + "\n")
	if kinds["main"] {
		b.WriteString("main\n")
	}
	if kinds["apps"] {
		b.WriteString("apps\n")
	}
	_ = os.WriteFile(p, []byte(b.String()), 0o600)
}
