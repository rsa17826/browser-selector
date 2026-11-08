package main

import (
	"fmt"
	"regexp"
	"strings"
)

func normalizePath(s string) string {
	return strings.ToLower(strings.Trim(strings.ReplaceAll(s, "\\", "/"), " /"))
}

// doTheyMatch returns an error for an unknown match type or a bad regex
// instead of treating it as "no match", so config mistakes are visible.
// Go uses RE2 syntax for matchesregex (no lookaheads/backreferences).
func doTheyMatch(value, matchType, pattern string) (bool, error) {
	switch matchType {
	case "is":
		return normalizePath(value) == normalizePath(pattern), nil
	case "isexact":
		return value == pattern, nil
	case "startswith":
		return strings.HasPrefix(value, pattern), nil
	case "endswith":
		return strings.HasSuffix(value, pattern), nil
	case "includes":
		return strings.Contains(value, pattern), nil
	case "matchesregex":
		re, err := regexp.Compile(pattern)
		if err != nil {
			return false, fmt.Errorf("bad regex %q: %w", pattern, err)
		}
		return re.MatchString(value), nil
	}
	return false, fmt.Errorf("invalid match type %q", matchType)
}
