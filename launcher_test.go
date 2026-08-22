package main

import "testing"

// The founder types "manymoats -- orch". It printed nothing at all, because "--"
// was read as the name of an app. "--" is also the POSIX end-of-options marker,
// so refusing it was wrong twice over.
func TestSeparatorAndDashedFormsReachTheApp(t *testing.T) {
	all := apps()
	for _, in := range [][]string{
		{"orch"},
		{"--", "orch"},
		{"--orch"},
		{"--", "credits", "--snapshot"},
		{"--credits"},
	} {
		got := normalise(in, all)
		if len(got) == 0 {
			t.Fatalf("%v resolved to nothing", in)
		}
		found := false
		for _, a := range all {
			if a.Name == got[0] {
				found = true
			}
		}
		if !found {
			t.Errorf("%v → %q, which is not an app", in, got[0])
		}
	}
	// a bare "--" with nothing after it is the directory, not an error
	if len(normalise([]string{"--"}, all)) != 0 {
		t.Error(`"--" alone should fall through to the directory`)
	}
}
