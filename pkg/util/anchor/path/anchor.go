// Package anchorpath provides utilities for manipulating anchor paths & components.
package anchorpath

import (
	"strings"
)

const sep = "/"

// Parts splits the anchor path into its constituent components
func Parts(path string) []string {
	var b strings.Builder
	b.Grow(len(path))

	parts := make([]string, 0, 8)
	for _, r := range path {
		if r == '/' {
			if b.Len() != 0 {
				parts = append(parts, b.String())
				b.Reset()
			}
			continue
		}

		b.WriteRune(r)
	}

	if b.Len() != 0 {
		parts = append(parts, b.String())
	}

	return parts
}

// Join path components
func Join(parts []string) string {
	return Clean(strings.Join(parts, sep))

}

// Clean the path through lexical analysis.
func Clean(path string) string {
	var b strings.Builder
	for _, part := range Parts(path) {
		b.WriteRune('/')
		b.WriteString(part)
	}

	if b.Len() == 0 {
		return sep
	}

	return b.String()
}

// Root returns true if the path points to the root anchor.
func Root(path []string) bool {
	if path == nil || len(path) == 0 {
		return true
	}

	return strings.Trim(path[0], sep) == ""
}
