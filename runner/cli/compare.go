package cli

import (
	"fmt"
	"strings"
)

const sharedEnvKey = "$shared"

func parseCompare(raw string) ([]string, error) {
	fields := compareFields(strings.TrimSpace(raw))
	if len(fields) == 0 {
		return nil, nil
	}

	seen := make(map[string]struct{}, len(fields))
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		name := strings.TrimSpace(field)
		if name == "" {
			continue
		}
		if strings.EqualFold(name, sharedEnvKey) {
			return nil, fmt.Errorf("environment %q is reserved for shared defaults", name)
		}

		key := strings.ToLower(name)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, name)
	}

	if len(out) < 2 {
		return nil, fmt.Errorf("expected at least two environments, got %d", len(out))
	}
	return out, nil
}

func compareFields(raw string) []string {
	if raw == "" {
		return nil
	}
	// Explicit separators allow spaces in environment names.
	if strings.ContainsAny(raw, ",;") {
		return strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ';' })
	}
	return strings.Fields(raw)
}
