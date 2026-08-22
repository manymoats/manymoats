package agent

import "testing"

func TestFallbackTriggersWhenAmbiguousIsWide(t *testing.T) {
	// Pin the icon set: this test must assert behaviour, not inherit whatever
	// font happens to be installed on the machine running it.
	t.Setenv("ORCH_ICONS", "unicode")
	resetMarks()
	defer func() { t.Setenv("ORCH_ICONS", ""); resetMarks() }()
	SetAmbiguousWide(false)
	// The board icon is the glyph itself — a frame is decoration, not size.
	if _, mid, _ := MarkFor(Claude).Render(); mid != "✶" {
		t.Fatalf("the board icon is the glyph, unframed — got %q", mid)
	}
	if MarkFor(Claude).Tiny() != "✶" {
		t.Fatalf("the one-cell form is for legends and the strip, got %q", MarkFor(Claude).Tiny())
	}
	SetAmbiguousWide(true)
	if _, mid, _ := MarkFor(Claude).Render(); mid != "*" {
		t.Fatalf("wide-ambiguous terminal must fall back to ASCII, got %q", mid)
	}
	if MarkFor(Claude).Tiny() != "*" {
		t.Fatalf("the one-cell form must fall back too, got %q", MarkFor(Claude).Tiny())
	}
	SetAmbiguousWide(false)
}

func TestEveryMarkHasAnASCIIFallback(t *testing.T) {
	for _, s := range []Source{Claude, Qwen, Cursor, Grok, Gemini, Ollama, Terminal, Muse} {
		if MarkFor(s).ASCII == "" {
			t.Errorf("%s has no ASCII fallback — it would vanish on a wide-ambiguous terminal", s)
		}
	}
}

func TestAmbiguousGlyphsAreKnownRisky(t *testing.T) {
	risky := map[Source]string{Qwen: "⬡", Cursor: "◱", Ollama: "◉", Grok: "╳"}
	for src, g := range risky {
		if !GlyphFits(g) {
			t.Logf("%s mark %q is not single-width by runewidth — ASCII fallback is load-bearing", src, g)
		}
		if MarkFor(src).ASCII == "" {
			t.Errorf("%s uses risky glyph %q with NO ascii fallback", src, g)
		}
	}
}
