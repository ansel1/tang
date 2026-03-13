// Package panicker provides functions that panic under certain conditions,
// used to demonstrate how tang displays test panics.
package panicker

import "fmt"

// ProcessItems iterates over items and panics on a nil entry.
func ProcessItems(items []string) []string {
	var result []string
	for i, item := range items {
		if item == "" {
			panic(fmt.Sprintf("empty item at index %d", i))
		}
		result = append(result, fmt.Sprintf("processed:%s", item))
	}
	return result
}

// BuildIndex creates a lookup map from a slice. Panics if a duplicate key is found.
func BuildIndex(keys []string, values []int) map[string]int {
	if len(keys) != len(values) {
		panic("keys and values must have the same length")
	}
	m := make(map[string]int, len(keys))
	for i, k := range keys {
		if _, exists := m[k]; exists {
			panic(fmt.Sprintf("duplicate key: %s", k))
		}
		m[k] = values[i]
	}
	return m
}

// RecursiveDepth recurses n levels deep and panics when depth reaches 0.
func RecursiveDepth(n int) int {
	if n <= 0 {
		panic("reached maximum recursion depth")
	}
	if n == 1 {
		return 1
	}
	return 1 + RecursiveDepth(n-1)
}
