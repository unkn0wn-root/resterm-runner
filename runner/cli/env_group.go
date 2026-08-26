package cli

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
)

type groupFlags map[string]string

func (f *groupFlags) Set(value string) error {
	group, profile, ok := strings.Cut(value, "=")
	group, profile = strings.TrimSpace(group), strings.TrimSpace(profile)
	if !ok || group == "" || profile == "" {
		return errors.New("expected group=profile")
	}

	// Map keys keep their original case, so a direct lookup is not enough.
	for name := range *f {
		if strings.EqualFold(name, group) {
			return fmt.Errorf("group %q selected more than once", group)
		}
	}

	if *f == nil {
		*f = make(groupFlags)
	}
	(*f)[group] = profile
	return nil
}

func (f groupFlags) String() string {
	names := slices.SortedFunc(maps.Keys(f), func(a, b string) int {
		return strings.Compare(strings.ToLower(a), strings.ToLower(b))
	})

	out := make([]string, len(names))
	for i, name := range names {
		out[i] = name + "=" + f[name]
	}
	return strings.Join(out, ",")
}
