// Package models provides shared data types. It has no tests,
// demonstrating how tang handles packages with no test files.
package models

// User represents a user account.
type User struct {
	ID    int
	Name  string
	Email string
	Role  string
}

// Item represents an inventory item.
type Item struct {
	ID    int
	Name  string
	Price float64
	Stock int
}
