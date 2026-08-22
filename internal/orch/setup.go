package orch

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/manymoats/manymoats/internal/agent"
)

// Setup is optional polish, never a gate. orch works with zero setup using plain
// shapes that render in every mono font on earth. This only exists for people who
// want the icons — and it asks before touching anything, because installing a
// font and editing a terminal config are both changes to someone else's machine.
func setup() int {
	in := bufio.NewReader(os.Stdin)
	ask := func(q string) bool {
		fmt.Printf("\n  %s [y/N] ", q)
		s, _ := in.ReadString('\n')
		return strings.HasPrefix(strings.ToLower(strings.TrimSpace(s)), "y")
	}

	fmt.Println("\n  orch icon setup                              by manymoats")
	fmt.Println("  ────────────────────────────────────────────────────────")
	fmt.Println("  orch already works. This only swaps the plain shapes for real")
	fmt.Println("  provider icons, which need a Nerd Font. Nothing here is required.")

	term := detectTerminal()
	fmt.Printf("\n  terminal:  %s\n", term.name)
	fmt.Printf("  nerd font: %v\n", agent.NerdFontInstalled())

	if !agent.NerdFontInstalled() {
		if _, err := exec.LookPath("brew"); err != nil {
			fmt.Println("\n  No Homebrew, so I can't install it for you. Get any Nerd Font from")
			fmt.Println("  https://nerdfonts.com and re-run this.")
			return 0
		}
		fmt.Println("\n  orch ships with JetBrainsMono Nerd Font as its default.")
		fmt.Println("  Would install:  brew install --cask font-jetbrains-mono-nerd-font")
		fmt.Println("  That adds font files to ~/Library/Fonts. Nothing else is touched.")
		if !ask("Install it?") {
			fmt.Println("\n  Left alone. orch keeps using plain shapes.")
			return 0
		}
		c := exec.Command("brew", "install", "--cask", "font-jetbrains-mono-nerd-font")
		c.Stdout, c.Stderr = os.Stdout, os.Stderr
		if err := c.Run(); err != nil {
			fmt.Println("\n  Install failed. orch keeps using plain shapes.")
			return 1
		}
	}

	if term.alreadySet() {
		fmt.Printf("\n  %s is already using %s — nothing to change.\n", term.name, strings.TrimSpace(term.current()))
	} else if term.canSet {
		fmt.Printf("\n  %s needs to be told to use it — terminals do not pick up\n", term.name)
		fmt.Println("  a new font on their own.")
		fmt.Printf("\n  Would set:  %s\n", term.change)
		if ask("Set it?") {
			if err := term.apply(); err != nil {
				fmt.Printf("\n  Could not set it: %v\n", err)
				fmt.Printf("  Do it by hand:  %s\n", term.manual)
			} else {
				fmt.Println("\n  Set. Open a NEW window — existing ones keep the old font.")
			}
		}
	} else {
		fmt.Printf("\n  Set your terminal's font by hand:\n    %s\n", term.manual)
	}

	// Somebody who just installed a font and pointed their terminal at it wants
	// the icons on. Asking again would be ceremony.
	if agent.NerdFontInstalled() && !agent.UseNerd() {
		if ask("Turn the icons on?") {
			setIcons(true)
		}
	} else {
		fmt.Printf("\n  icons are currently: %s\n", map[bool]string{true: "nerd", false: "plain"}[agent.UseNerd()])
	}

	fmt.Println("\n  Then check with:  orch --icons")
	fmt.Println("  Boxes instead of icons means the font is not reaching the terminal.")
	fmt.Println("  Turn them on:     orch --icons-on      (or ORCH_ICONS=nerd orch)")
	fmt.Println("  Turn them off:    orch --icons-off")
	fmt.Println()
	return 0
}

type terminalInfo struct {
	name, change, manual string
	canSet               bool
	apply                func() error
	current              func() string // what font the terminal is on now, "" if unknowable
}

// alreadySet reports whether the terminal is already using a Nerd Font, so
// running setup twice says "done" instead of offering to redo work.
func (t terminalInfo) alreadySet() bool {
	if t.current == nil {
		return false
	}
	cur := strings.ToLower(t.current())
	return strings.Contains(cur, "nerd") || strings.Contains(cur, "nfm") ||
		strings.Contains(cur, " nf")
}

func detectTerminal() terminalInfo {
	const family = "'JetBrainsMono Nerd Font Mono', 'JetBrainsMono NFM', monospace"
	switch os.Getenv("TERM_PROGRAM") {
	case "Apple_Terminal":
		return terminalInfo{
			name: "Terminal.app", canSet: true,
			change: "font of every Terminal profile → JetBrainsMonoNFM-Regular",
			manual: "Terminal → Settings → Profiles → Font → JetBrainsMono Nerd Font Mono",
			current: func() string {
				out, err := exec.Command("osascript", "-e",
					`tell application "Terminal" to return font name of settings set (name of default settings)`).Output()
				if err != nil {
					return ""
				}
				return strings.TrimSpace(string(out))
			},
			apply: func() error {
				return exec.Command("osascript", "-e", `tell application "Terminal"
  repeat with s in settings sets
    try
      set font name of s to "JetBrainsMonoNFM-Regular"
    end try
  end repeat
end tell`).Run()
			},
		}
	case "vscode":
		p := vscodeSettings()
		return terminalInfo{
			name: "VS Code / Cursor terminal", canSet: p != "",
			change: "terminal.integrated.fontFamily → " + family,
			manual: `Settings → "terminal font family" → ` + family,
			apply:  func() error { return setJSONKey(p, "terminal.integrated.fontFamily", family) },
			current: func() string {
				b, err := os.ReadFile(p)
				if err != nil {
					return ""
				}
				i := strings.Index(string(b), `"terminal.integrated.fontFamily"`)
				if i < 0 {
					return ""
				}
				rest := string(b)[i:]
				if j := strings.Index(rest, "\n"); j > 0 {
					return rest[:j]
				}
				return ""
			},
		}
	case "iTerm.app":
		return terminalInfo{name: "iTerm2",
			manual: "iTerm2 → Settings → Profiles → Text → Font → JetBrainsMono Nerd Font Mono"}
	case "ghostty":
		return terminalInfo{name: "Ghostty",
			manual: `add to ~/.config/ghostty/config:  font-family = JetBrainsMono Nerd Font Mono`}
	case "WezTerm":
		return terminalInfo{name: "WezTerm",
			manual: `in ~/.wezterm.lua:  font = wezterm.font("JetBrainsMono Nerd Font Mono")`}
	}
	return terminalInfo{name: "unknown (" + os.Getenv("TERM_PROGRAM") + ")",
		manual: "set your terminal's font to: JetBrainsMono Nerd Font Mono"}
}

func vscodeSettings() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	for _, app := range []string{"Cursor", "Code", "VSCodium"} {
		p := filepath.Join(home, "Library", "Application Support", app, "User", "settings.json")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// setJSONKey edits one key in a JSONC settings file, preserving everything else
// and backing the original up first. It is somebody else's editor config.
func setJSONKey(path, key, val string) error {
	if path == "" {
		return fmt.Errorf("no settings file found")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path+".orch-backup", raw, 0o600); err != nil {
		return err
	}
	s := string(raw)
	needle := `"` + key + `"`
	if i := strings.Index(s, needle); i >= 0 {
		// replace just this key's value, leaving comments and order intact
		colon := strings.Index(s[i:], ":")
		if colon < 0 {
			return fmt.Errorf("malformed settings")
		}
		start := i + colon + 1
		end := start
		for end < len(s) && s[end] != ',' && s[end] != '\n' && s[end] != '}' {
			end++
		}
		s = s[:start] + ` "` + strings.ReplaceAll(val, `"`, `\"`) + `"` + s[end:]
	} else {
		j := strings.LastIndex(s, "}")
		if j < 0 {
			return fmt.Errorf("malformed settings")
		}
		insert := "  " + needle + `: "` + strings.ReplaceAll(val, `"`, `\"`) + `"`
		before := strings.TrimRight(s[:j], " \t\r\n")
		if !strings.HasSuffix(before, "{") {
			before += ","
		}
		s = before + "\n" + insert + "\n" + s[j:]
	}
	return os.WriteFile(path, []byte(s), 0o600)
}
