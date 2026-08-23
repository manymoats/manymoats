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

func TestAWideGlyphFallsBackToASCII(t *testing.T) {
	SetAmbiguousWide(false)
	m := Mark{Glyph: "🦙", ASCII: "*", Mid: "🦙"}
	if _, mid, _ := m.Render(); mid != "*" {
		t.Fatalf("a 2-cell emoji must fall back to ASCII, got %q", mid)
	}
	if m.Tiny() != "*" {
		t.Fatalf("Tiny must fall back too, got %q", m.Tiny())
	}
	if Width("🦙") != 2 {
		t.Fatalf("llama is two cells, got %d — cells are not bytes", Width("🦙"))
	}
	if GlyphFits("🦙") {
		t.Fatal("a 2-cell glyph must not be called a fit")
	}
	if !GlyphFits("*") {
		t.Fatal("ASCII star is one cell")
	}
	if !GlyphFits("▁") {
		t.Fatal("waveform bars must be one cell or the board will drift")
	}
}

func TestToolIconsAreHonestWhenNerdIsOff(t *testing.T) {
	t.Setenv("ORCH_ICONS", "plain")
	resetMarks()
	defer func() { t.Setenv("ORCH_ICONS", ""); resetMarks() }()
	if got := Icon("play"); got != ">" {
		t.Fatalf("without nerd, play is >, got %q", got)
	}
	if got := Icon("check"); got != "ok" {
		t.Fatalf("without nerd, check is ok, got %q", got)
	}
	for _, ic := range AllToolIcons() {
		g := Icon(ic.Name)
		if !GlyphFits(g) && Width(g) > 2 {
			t.Errorf("%s fallback %q is too wide (%d cells)", ic.Name, g, Width(g))
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
