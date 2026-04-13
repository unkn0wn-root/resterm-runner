package cli

import (
	"fmt"
	"strings"
)

const sharedEnvKey = "$shared"

func parseCompare(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	repl := strings.NewReplacer(",", " ", ";", " ")
	fields := strings.Fields(repl.Replace(raw))
	if len(fields) == 0 {
		return nil, nil
	}

	seen := make(map[string]struct{}, len(fields))
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if isReservedEnv(field) {
			return nil, fmt.Errorf("environment %q is reserved for shared defaults", field)
		}
		key := strings.ToLower(field)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, field)
	}
	if len(out) < 2 {
		return nil, fmt.Errorf("expected at least two environments, got %d", len(out))
	}
	return out, nil
}

func isReservedEnv(name string) bool {
	return strings.EqualFold(strings.TrimSpace(name), sharedEnvKey)
}
