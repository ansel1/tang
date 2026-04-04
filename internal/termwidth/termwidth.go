// Package termwidth resolves terminal width with support for the COLUMNS
// environment variable.
//
// Resolution order:
//  1. COLUMNS environment variable (hard override, per Unix convention)
//  2. Terminal ioctl detection via the given file descriptor
//  3. Fallback default (80)
package termwidth

import (
	"os"
	"strconv"

	"github.com/charmbracelet/x/term"
)

const DefaultWidth = 80

// Get returns the terminal width. If COLUMNS is set to a positive integer,
// it takes priority over ioctl detection. Otherwise the width is read from
// the given file descriptor (typically os.Stdout). If detection fails, 80
// is returned.
func Get(fd uintptr) int {
	if cols := fromEnv(); cols > 0 {
		return cols
	}
	if w, _, err := term.GetSize(fd); err == nil && w > 0 {
		return w
	}
	return DefaultWidth
}

// FromEnv returns the value of the COLUMNS environment variable, or 0 if
// it is unset or not a valid positive integer.
func FromEnv() int {
	return fromEnv()
}

func fromEnv() int {
	s := os.Getenv("COLUMNS")
	if s == "" {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}
