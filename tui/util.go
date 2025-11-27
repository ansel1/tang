package tui

import "strings"

// ensureReset ensures that the string ends with a terminal reset sequence.
// This prevents color bleeding from truncated output or output that leaves colors open.
func ensureReset(s string) string {
	if s == "" {
		return ""
	}
	// If the string already ends with a reset sequence, don't add another one
	if strings.HasSuffix(s, "\033[0m") {
		return s
	}
	return s + "\033[0m"
}
