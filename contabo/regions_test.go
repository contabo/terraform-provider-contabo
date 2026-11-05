package contabo

import "testing"

func TestNormalizeRegion(t *testing.T) {
	cases := map[string]string{
		"EU":                      "EU",
		"European Union":          "EU",
		"european union":          "EU",
		"United States (Central)": "US-central",
		"us-central":              "US-central",
		"Asia (Singapore)":        "SIN",
		"Unknown Region":          "Unknown Region",
	}
	for in, want := range cases {
		if got := normalizeRegion(in); got != want {
			t.Errorf("normalizeRegion(%q) = %q, want %q", in, got, want)
		}
	}
	if !suppressEquivalentRegion("", "European Union", "EU", nil) {
		t.Error("European Union and EU should be equivalent")
	}
	if suppressEquivalentRegion("", "European Union", "SIN", nil) {
		t.Error("European Union and SIN should differ")
	}
}
