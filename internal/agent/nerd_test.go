package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAnyNerdFontCounts(t *testing.T) {
	d := t.TempDir()
	t.Setenv("HOME", d)
	fonts := filepath.Join(d, "Library", "Fonts")
	if err := os.MkdirAll(fonts, 0o755); err != nil {
		t.Fatal(err)
	}
	if NerdFontInstalled() {
		t.Fatal("empty font dir must report false")
	}
	// a patched font that is NOT the symbols-only build
	os.WriteFile(filepath.Join(fonts, "JetBrainsMonoNerdFont-Regular.ttf"), []byte("x"), 0o644)
	if !NerdFontInstalled() {
		t.Fatal("any Nerd Font must count — looking for one filename reported false on a machine that had one")
	}
}
