package cli

import (
	"fmt"
	"strings"
)

const sharedEnvKey = "$shared"

func parseCompare(raw string) ([]string, error) {
	raw = trim(raw)
	if raw == "" {
		return nil, nil
	}

	r := strings.NewReplacer(",", " ", ";", " ")
	fd := strings.Fields(r.Replace(raw))
	if len(fd) == 0 {
		return nil, nil
	}

	seen := make(map[string]struct{}, len(fd))
	out := make([]string, 0, len(fd))
	for _, f := range fd {
		if isReservedEnv(f) {
			return nil, fmt.Errorf("environment %q is reserved for shared defaults", f)
		}
		key := strings.ToLower(f)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, f)
	}
	if len(out) < 2 {
		return nil, fmt.Errorf("expected at least two environments, got %d", len(out))
	}
	return out, nil
}

func isReservedEnv(name string) bool {
	return strings.EqualFold(trim(name), sharedEnvKey)
}
