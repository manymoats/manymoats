package agent

// Tool icons. Status only — never mascots, never vendor logos, never emoji.
// Nerd glyphs are used only when the user asked for them AND the glyph is one
// cell. Anything else is the ASCII next to it, which always lines up.

type ToolIcon struct {
	Name  string
	Nerd  string
	ASCII string
}

var toolIcons = []ToolIcon{
	{Name: "play", Nerd: "\uF04B", ASCII: ">"},
	{Name: "pause", Nerd: "\uF04C", ASCII: "||"},
	{Name: "check", Nerd: "\uF00C", ASCII: "ok"},
	{Name: "x", Nerd: "\uF00D", ASCII: "x"},
	{Name: "arrow", Nerd: "\uF061", ASCII: ">"},
	{Name: "folder", Nerd: "\uF07B", ASCII: "[]"},
	{Name: "git", Nerd: "\uF126", ASCII: "Y"},
	{Name: "circle", Nerd: "\uF111", ASCII: "*"},
	{Name: "pulse", Nerd: "\uF21E", ASCII: "*"},
}

// Icon returns a status glyph that will occupy a known number of cells.
func Icon(name string) string {
	for _, ic := range toolIcons {
		if ic.Name != name {
			continue
		}
		if UseNerd() && NerdFontInstalled() && GlyphFits(ic.Nerd) {
			return ic.Nerd
		}
		return ic.ASCII
	}
	return "*"
}

func AllToolIcons() []ToolIcon { return toolIcons }
