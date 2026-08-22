package agent

import "testing"

func TestNeverAnEllipsis(t *testing.T) {
	cases := []string{"qwen3-embedding:0.6b", "llama3.1:latest", "claude-opus-5", "manymoats-kpf",
		"nemotron-3-super-120b-a12b", "some/very/long/model/path/name"}
	for _, c := range cases {
		for _, n := range []int{4, 8, 10, 12} {
			got := Shorten(c, n)
			if containsAny(got, "…", "...") {
				t.Errorf("Shorten(%q,%d) = %q — an ellipsis spends a character to say a character is missing", c, n, got)
			}
			if len([]rune(got)) > n {
				t.Errorf("Shorten(%q,%d) = %q, too long", c, n, got)
			}
		}
	}
}

func TestKeepsWhatIdentifies(t *testing.T) {
	for _, c := range []struct {
		in, want string
		n        int
	}{
		{"qwen3-embedding:0.6b", "qwen3", 10},
		{"llama3.1:latest", "llama3.1", 10},
		{"claude-opus-5", "claude-opus-5", 14},
		{"nemotron-3-super-120b-a12b", "nemotron-3", 12},
	} {
		if got := Shorten(c.in, c.n); got != c.want {
			t.Errorf("Shorten(%q,%d) = %q, want %q", c.in, c.n, got, c.want)
		}
	}
}

func TestProjectDropsHouseSuffixes(t *testing.T) {
	if got := ShortProject("manymoats-kpf", 12); got != "manymoats" {
		t.Fatalf("got %q want manymoats", got)
	}
	if got := ShortProject("manyAPPS-kpf", 12); got != "manyAPPS" {
		t.Fatalf("got %q want manyAPPS", got)
	}
}

func containsAny(s string, subs ...string) bool {
	for _, x := range subs {
		for i := 0; i+len(x) <= len(s); i++ {
			if s[i:i+len(x)] == x {
				return true
			}
		}
	}
	return false
}
