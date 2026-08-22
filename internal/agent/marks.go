package agent

import (
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

//go:embed data/marks.json
var marksJSON []byte

//go:embed data/marks-nerd.json
var marksNerdJSON []byte

// UseNerd is OPT-IN ONLY, and deliberately so.
//
// The first version auto-enabled when the font FILE existed. That was wrong for
// the same reason reading codesign's silence as a pass was wrong: a file being
// present is not the same as a glyph rendering. Terminal emulators built on
// xterm.js — Cursor's and VS Code's — do not inherit macOS font fallback, so an
// installed Nerd Font goes unused unless the terminal's own fontFamily names it.
// Guessing wrong here costs a board of empty squares, which is worse than plain
// shapes that always draw. So: the user turns it on, after looking.
func UseNerd() bool {
	switch os.Getenv("ORCH_ICONS") {
	case "nerd":
		return true
	case "unicode", "plain":
		return false
	}
	// Config decides when the environment does not. Still never auto-detected
	// from a font file existing — see the comment above.
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	b, err := os.ReadFile(filepath.Join(home, ".orch", "config.json"))
	if err != nil {
		return false
	}
	var c struct {
		Icons string `json:"icons"`
	}
	if json.Unmarshal(b, &c) != nil {
		return false
	}
	return c.Icons == "nerd"
}

// NerdFontInstalled reports only that a patched font is on disk — never that it
// will render. It matches ANY Nerd Font, not one particular filename: the first
// version looked only for the symbols-only build and reported "false" on a
// machine that had JetBrainsMono Nerd Font sitting right there.
func NerdFontInstalled() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	for _, dir := range []string{
		filepath.Join(home, "Library", "Fonts"),
		"/Library/Fonts",
	} {
		ents, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range ents {
			n := strings.ToLower(e.Name())
			if strings.Contains(n, "nerdfont") || strings.Contains(n, "nerd font") {
				return true
			}
		}
	}
	return false
}

type Mark struct {
	ID    string   `json:"id"`
	Label string   `json:"label"`
	Glyph string   `json:"glyph"`
	ASCII string   `json:"ascii"`
	Color string   `json:"colour"`
	Top   string   `json:"top"`
	Mid   string   `json:"mid"`
	Bot   string   `json:"bot"`
	Match []string `json:"match"`
}

var (
	loadOnce sync.Once
	marks    map[Source]Mark
	allMarks []Mark
)

func load() {
	var list []Mark
	if json.Unmarshal(marksJSON, &list) != nil {
		list = nil
	}
	if UseNerd() {
		var nerd []Mark
		if json.Unmarshal(marksNerdJSON, &nerd) == nil {
			over := map[string]Mark{}
			for _, n := range nerd {
				over[n.ID] = n
			}
			for i, m := range list {
				if n, ok := over[m.ID]; ok {
					list[i].Glyph = n.Glyph
					// Big() reads Mid. Leaving it on the unicode set meant the
					// splash and the board showed the same source two ways.
					list[i].Mid = n.Glyph
					if n.Top != "" {
						list[i].Top = n.Top
					}
					if n.Bot != "" {
						list[i].Bot = n.Bot
					}
				}
			}
		}
	}
	// A user file wins, so anyone can add a provider without rebuilding.
	if home, err := os.UserHomeDir(); err == nil {
		if b, err := os.ReadFile(filepath.Join(home, ".orch", "marks.json")); err == nil {
			var extra []Mark
			if json.Unmarshal(b, &extra) == nil {
				list = append(list, extra...)
			}
		}
	}
	marks = make(map[Source]Mark, len(list))
	for _, m := range list {
		marks[Source(m.ID)] = m
	}
	allMarks = list
}

// resetMarks forces a reload, so a test can pin the icon set instead of
// inheriting whatever font happens to be installed on the machine running it.
func resetMarks() {
	loadOnce = sync.Once{}
}

func MarkFor(s Source) Mark {
	loadOnce.Do(load)
	if m, ok := marks[s]; ok {
		return m
	}
	return Mark{Glyph: "·", ASCII: ".", Color: "#6b717a", Top: "   ", Mid: " · ", Bot: "   "}
}

// SourceOf resolves a model id or process name to a provider, so a new model
// from a known vendor is recognised without a code change.
func SourceOf(text string) (Source, bool) {
	loadOnce.Do(load)
	t := strings.ToLower(text)
	for _, m := range allMarks {
		for _, k := range m.Match {
			if strings.Contains(t, strings.ToLower(k)) {
				return Source(m.ID), true
			}
		}
	}
	return "", false
}

func AllMarks() []Mark {
	loadOnce.Do(load)
	return allMarks
}
