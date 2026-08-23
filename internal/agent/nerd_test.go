package agent

import (
	"encoding/json"
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

// The nerd overlay exists because a glyph the font lacks renders as a blank box.
// Shipping an emoji in it defeats the whole point, so the set stays inside the
// private-use planes the Nerd Font patch actually fills.
func TestNerdGlyphsStayInThePatchedRanges(t *testing.T) {
	for _, m := range nerdMarks(t) {
		for _, r := range m.Glyph {
			inPUA := (r >= 0xE000 && r <= 0xF8FF) || (r >= 0xF0000 && r <= 0xFFFFD)
			if !inPUA && r > 0x2FFF {
				t.Errorf("%s ships %q (U+%04X) in the nerd set — outside the patched ranges", m.ID, m.Glyph, r)
			}
		}
	}
}

func TestALinuxFontDirCounts(t *testing.T) {
	d := t.TempDir()
	t.Setenv("HOME", d)
	fonts := filepath.Join(d, ".local", "share", "fonts", "truetype")
	if err := os.MkdirAll(fonts, 0o755); err != nil {
		t.Fatal(err)
	}
	if NerdFontInstalled() {
		t.Fatal("empty linux font dir must report false")
	}
	os.WriteFile(filepath.Join(fonts, "JetBrainsMonoNerdFont-Regular.ttf"), []byte("x"), 0o644)
	if !NerdFontInstalled() {
		t.Fatal("a Nerd Font under ~/.local/share/fonts must count")
	}
}

func TestNerdOverlayRefusesEmoji(t *testing.T) {
	if nerdGlyphOK("🦙") {
		t.Fatal("an emoji must not be treated as a usable nerd glyph")
	}
	if nerdGlyphOK("✶") {
		t.Fatal("a unicode mark is not a nerd glyph — that path is PUA only")
	}
	if !nerdGlyphOK("\uF111") {
		t.Fatal("a 1-cell PUA circle must be accepted")
	}
}

func nerdMarks(t *testing.T) []Mark {
	t.Helper()
	var ms []Mark
	if err := json.Unmarshal(marksNerdJSON, &ms); err != nil {
		t.Fatal(err)
	}
	return ms
}
