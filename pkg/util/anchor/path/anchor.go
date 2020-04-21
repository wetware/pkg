// Package anchorpath provides utilities for manipulating anchor paths & components.
package anchorpath

import (
	"path/filepath"
	"strings"
)

const (
	sep = "/"
)

// Clean anchor path via pure lexical analysis
func Clean(p string) string {
	return filepath.Clean(p)
}

// Parts splits the anchor path into its constituent components
func Parts(p string) []string {
	raw := strings.Split(strings.Trim(Clean(p), sep), sep)

	res := raw[:0]
	for _, p = range raw {
		if p == "" {
			continue
		}

		res = append(res, p)
	}

	return res
}

// Join path components
func Join(p ...string) string {
	return Clean(filepath.Join(p...))
}

// Rootify prepends a "/" to the specified path, if it is missing.
func Rootify(p string) string {
	return Clean(filepath.Join(sep, p))
}

// Abs returns true if the path is absolute.
// It is identical to filepath.IsAbs, and is included for convenience.
func Abs(p string) bool {
	return strings.HasPrefix(p, "/")
}
