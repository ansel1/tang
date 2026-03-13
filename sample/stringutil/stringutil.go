package stringutil

import "strings"

// Reverse returns the string reversed.
func Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// Palindrome reports whether s reads the same forwards and backwards.
func Palindrome(s string) bool {
	s = strings.ToLower(s)
	return s == Reverse(s)
}

// WordCount returns the number of whitespace-separated words.
func WordCount(s string) int {
	return len(strings.Fields(s))
}

// Truncate shortens s to maxLen characters, appending "..." if truncated.
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
