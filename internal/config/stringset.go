package config

import (
	"slices"
	"strings"
)

// StringSet is a [flag.Value] that collects one ordered, deduplicated string
// per flag occurrence.
type StringSet []string

func (s *StringSet) String() string {
	if s == nil {
		return ""
	}
	return strings.Join(*s, ",")
}

// Set ignores empty values so `--gateway-name=` doesn't silently widen the
// filter to "any name".
func (s *StringSet) Set(v string) error {
	if v == "" || slices.Contains(*s, v) {
		return nil
	}
	*s = append(*s, v)
	return nil
}

func (s StringSet) Contains(v string) bool { return slices.Contains(s, v) }
