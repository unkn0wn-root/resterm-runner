package cli

import (
	"runtime/debug"
	"testing"
)

func TestFormatBuildTime(t *testing.T) {
	for in, want := range map[string]string{
		"":                          "",
		"garbage":                   "garbage",
		"2026-04-14T10:20:30Z":      "2026-04-14 10:20:30 UTC",
		"2026-04-14T12:20:30+02:00": "2026-04-14 10:20:30 UTC",
	} {
		if got := formatBuildTime(in); got != want {
			t.Fatalf("formatBuildTime(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRuntimeBuildMeta(t *testing.T) {
	prev := readBuildInfo
	defer func() { readBuildInfo = prev }()

	for _, tc := range []struct {
		name string
		fn   func() (*debug.BuildInfo, bool)
		want buildMeta
	}{
		{"not ok", func() (*debug.BuildInfo, bool) { return nil, false }, buildMeta{}},
		{"nil info but ok", func() (*debug.BuildInfo, bool) { return nil, true }, buildMeta{}},
		{"empty version", func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{}, true
		}, buildMeta{}},
		{"devel version", func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, true
		}, buildMeta{}},
		{"whitespace version", func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{Main: debug.Module{Version: "   "}}, true
		}, buildMeta{}},
		{"dirty long rev", func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{
				Main: debug.Module{Version: " v1.0.0 "},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: " 5a24f307ff0123 "},
					{Key: "vcs.modified", Value: "true"},
					{Key: "vcs.time", Value: "2026-04-14T10:20:30Z"},
				},
			}, true
		}, buildMeta{Version: "v1.0.0", Commit: "5a24f30-dirty", Date: "2026-04-14 10:20:30 UTC"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			readBuildInfo = tc.fn
			if got := runtimeBuildMeta(); got != tc.want {
				t.Fatalf("runtimeBuildMeta() = %+v, want %+v", got, tc.want)
			}
		})
	}
}
