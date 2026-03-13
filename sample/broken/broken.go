// Package broken has a deliberate compile error to demonstrate
// how tang displays build failures.
package broken

func Add(a, b int) int {
	return a +
}
