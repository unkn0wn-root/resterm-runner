package cli

import "strings"

func (o Opt) trimmed() Opt {
	o.Use = trim(o.Use)
	o.Version = trim(o.Version)
	o.Commit = trim(o.Commit)
	o.Date = trim(o.Date)
	return o
}

func trim(value string) string {
	return strings.TrimSpace(value)
}

func isBlank(value string) bool {
	return trim(value) == ""
}
