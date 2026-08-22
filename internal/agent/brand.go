package agent

import "strings"

type NameMode int

const (
	NameModel NameMode = iota
	NameBrand
	NameBoth
)

func Brand(s Source) string {
	if m := MarkFor(s); m.Label != "" {
		return m.Label
	}
	return string(s)
}

// ModelName strips the vendor prefix a model id carries, because the MARK already
// says who made it. "claude-opus-5" beside a Claude mark repeats itself.
func ModelName(model string, src Source) string {
	if model == "" {
		return "—"
	}
	m := model
	for _, p := range []string{"claude-", "anthropic/", "openai/", "google/", "qwen/", "moonshotai/", "deepseek/"} {
		m = strings.TrimPrefix(m, p)
	}
	if i := strings.LastIndex(m, "/"); i >= 0 {
		m = m[i+1:]
	}
	return m
}

func Display(a Agent, mode NameMode) string {
	switch mode {
	case NameBrand:
		return Brand(a.Source)
	case NameBoth:
		return Brand(a.Source) + " " + ModelName(a.Model, a.Source)
	default:
		return ModelName(a.Model, a.Source)
	}
}
