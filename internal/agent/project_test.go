package agent

import "testing"

func TestOneProjectNeedsNoLabels(t *testing.T) {
	as := []Agent{{Project: "kpf"}, {Project: "kpf"}, {Project: "kpf"}}
	if !SingleProject(as) {
		t.Fatal("all in one project — labelling every row repeats the same word three times")
	}
}

func TestSeveralProjectsGroupInAStableOrder(t *testing.T) {
	as := []Agent{{Project: "b"}, {Project: "a"}, {Project: "b"}, {Project: "c"}}
	order, by := GroupByProject(as)
	if len(order) != 3 || order[0] != "b" || order[1] != "a" {
		t.Fatalf("order must follow first appearance, got %v", order)
	}
	if len(by["b"]) != 2 {
		t.Fatalf("grouping lost an agent: %v", by)
	}
}

func TestProjectHueIsStableAndDistinct(t *testing.T) {
	if ProjectHue("kpf") != ProjectHue("kpf") {
		t.Fatal("a project's colour must not change between refreshes")
	}
	seen := map[string]string{}
	for _, p := range []string{"manyAPPS", "manymoats", "orch", "filefriend", "lathe"} {
		h := ProjectHue(p)
		if other, clash := seen[h]; clash {
			t.Logf("note: %s and %s share hue %s", p, other, h)
		}
		seen[h] = p
	}
	for _, ph := range projectHues {
		for _, m := range marks {
			if m.Color == ph {
				t.Errorf("project hue %s collides with a provider hue — a folder must never look like an agent", ph)
			}
		}
	}
}
