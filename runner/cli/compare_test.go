package cli

import (
	"slices"
	"testing"
)

func TestParseCompareSupportsProfilesWithSpaces(t *testing.T) {
	got, err := parseCompare("dev app 1, dev app 2")
	if err != nil {
		t.Fatalf("parseCompare: %v", err)
	}
	want := []string{"dev app 1", "dev app 2"}
	if !slices.Equal(got, want) {
		t.Fatalf("parseCompare() = %q, want %q", got, want)
	}
}

func TestParseCompareKeepsWhitespaceSeparatedNames(t *testing.T) {
	got, err := parseCompare("dev stage")
	if err != nil {
		t.Fatalf("parseCompare: %v", err)
	}
	want := []string{"dev", "stage"}
	if !slices.Equal(got, want) {
		t.Fatalf("parseCompare() = %q, want %q", got, want)
	}
}

func TestParseCompareIgnoresBlankEntries(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    []string
		wantErr string
	}{
		{name: "blank between names", raw: "dev, ,prod", want: []string{"dev", "prod"}},
		{name: "tab between names", raw: "dev,\t,prod", want: []string{"dev", "prod"}},
		{name: "repeated blanks", raw: "dev, , ,prod", want: []string{"dev", "prod"}},
		{name: "trailing blank leaves one name", raw: "dev, ,", wantErr: "expected at least two environments, got 1"},
		{name: "only blanks", raw: " , ", want: nil},
		{name: "unset", raw: "", want: nil},
		{name: "adjacent separators", raw: "dev,,prod", want: []string{"dev", "prod"}},
		{name: "semicolons", raw: "dev; ;prod", want: []string{"dev", "prod"}},
		{name: "names keep inner spaces", raw: "dev app 1, ,dev app 2", want: []string{"dev app 1", "dev app 2"}},
		{name: "reserved still rejected", raw: "dev, ,$shared", wantErr: `environment "$shared" is reserved for shared defaults`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCompare(tt.raw)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("parseCompare(%q) error = %v, want %q", tt.raw, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseCompare(%q): %v", tt.raw, err)
			}
			if !slices.Equal(got, tt.want) {
				t.Fatalf("parseCompare(%q) = %q, want %q", tt.raw, got, tt.want)
			}
			if slices.Contains(got, "") {
				t.Fatalf("parseCompare(%q) leaked an empty environment name: %q", tt.raw, got)
			}
		})
	}
}
