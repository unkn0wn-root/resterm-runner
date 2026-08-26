package cli

import (
	"strings"
	"testing"
)

func TestGroupFlags(t *testing.T) {
	var flags groupFlags
	if err := flags.Set(" api = prod "); err != nil {
		t.Fatalf("Set api: %v", err)
	}
	if err := flags.Set("app=dev app 2"); err != nil {
		t.Fatalf("Set app: %v", err)
	}
	if got := flags["api"]; got != "prod" {
		t.Fatalf("api = %q, want prod", got)
	}
	if got := flags["app"]; got != "dev app 2" {
		t.Fatalf("app = %q, want dev app 2", got)
	}
	if got, want := flags.String(), "api=prod,app=dev app 2"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestGroupFlagsRejectInvalidAndDuplicateValues(t *testing.T) {
	tests := []string{"", "api", "=prod", "api="}
	for _, value := range tests {
		t.Run(value, func(t *testing.T) {
			var flags groupFlags
			if err := flags.Set(value); err == nil || !strings.Contains(err.Error(), "group=profile") {
				t.Fatalf("Set(%q) error = %v, want group=profile error", value, err)
			}
		})
	}

	flags := groupFlags{"api": "dev"}
	if err := flags.Set("API=prod"); err == nil || !strings.Contains(err.Error(), "more than once") {
		t.Fatalf("duplicate Set error = %v, want duplicate error", err)
	}
}
