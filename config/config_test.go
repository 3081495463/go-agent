package config

import "testing"

func TestSplitArgs(t *testing.T) {
	cases := map[string][]string{
		"":                                  nil,
		"  ":                                nil,
		"@playwright/mcp@latest --headless": {"@playwright/mcp@latest", "--headless"},
	}

	for raw, want := range cases {
		got := splitArgs(raw)
		if len(got) != len(want) {
			t.Fatalf("splitArgs(%q) = %v, want %v", raw, got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("splitArgs(%q)[%d] = %q, want %q", raw, i, got[i], want[i])
			}
		}
	}
}
