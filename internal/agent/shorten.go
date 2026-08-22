package agent

import "strings"

// Shorten drops what is noise and keeps what identifies. It NEVER appends an
// ellipsis — an ellipsis spends a character to tell you a character is missing,
// which is worse than just showing fewer characters.
func Shorten(s string, max int) string {
	if s == "" {
		return "—"
	}
	// tags like :latest, :0.6b, :q4_K_M carry no identity at a glance
	if i := strings.IndexByte(s, ':'); i > 0 {
		s = s[:i]
	}
	if len([]rune(s)) <= max {
		return s
	}
	// drop trailing qualifiers at a separator rather than mid-word
	for _, sep := range []byte{'-', '_', '.', '/'} {
		for {
			i := strings.LastIndexByte(s, sep)
			if i <= 0 {
				break
			}
			s = s[:i]
			if len([]rune(s)) <= max {
				return s
			}
		}
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}

// ShortProject drops the suffixes every repo in this house shares.
func ShortProject(p string, max int) string {
	if p == "" || p == UnknownProject {
		return "—"
	}
	for _, suf := range []string{"-kpf", "-repo", ".git", "-main", "-app"} {
		p = strings.TrimSuffix(p, suf)
	}
	return Shorten(p, max)
}
