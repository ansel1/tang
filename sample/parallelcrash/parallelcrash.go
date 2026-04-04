// Package parallelcrash demonstrates how tang handles unrecoverable runtime
// fatals (like concurrent map writes) inside parallel tests while sibling
// parallel tests are still running.
package parallelcrash

import "fmt"

func Transform(items []*string) []string {
	out := make([]string, len(items))
	for i, p := range items {
		if p == nil {
			panic(fmt.Sprintf("nil pointer at index %d", i))
		}
		out[i] = fmt.Sprintf("transformed:%s", *p)
	}
	return out
}

func Merge(a, b []string) []string {
	if len(a) != len(b) {
		panic(fmt.Sprintf("length mismatch: %d vs %d", len(a), len(b)))
	}
	out := make([]string, len(a))
	for i := range a {
		out[i] = a[i] + "+" + b[i]
	}
	return out
}

// ConcurrentMapWrites triggers a "fatal error: concurrent map writes" by
// racing goroutines writing to an unprotected map. This is an unrecoverable
// runtime throw — the process exits immediately with no chance for test
// harness cleanup.
func ConcurrentMapWrites() {
	m := map[int]int{}
	done := make(chan struct{})
	for i := range 2 {
		go func() {
			for j := range 1_000_000 {
				m[i*1_000_000+j] = j
			}
			done <- struct{}{}
		}()
	}
	<-done
	<-done
}
