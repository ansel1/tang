package auth

import (
	"errors"
	"strings"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
	ErrWeakPassword       = errors.New("password too weak")
	ErrAccountLocked      = errors.New("account is locked")
)

// User holds authentication details.
type User struct {
	Username string
	Password string
	Locked   bool
}

// Store is an in-memory user store.
type Store struct {
	users map[string]User
}

// NewStore creates an empty user store.
func NewStore() *Store {
	return &Store{users: make(map[string]User)}
}

// Register adds a new user. Returns ErrWeakPassword if the password
// is shorter than 8 characters.
func (s *Store) Register(username, password string) error {
	if len(password) < 8 {
		return ErrWeakPassword
	}
	s.users[username] = User{
		Username: username,
		Password: password,
	}
	return nil
}

// Authenticate checks credentials. Returns nil on success.
func (s *Store) Authenticate(username, password string) error {
	u, ok := s.users[username]
	if !ok {
		return ErrUserNotFound
	}
	if u.Locked {
		return ErrAccountLocked
	}
	if u.Password != password {
		return ErrInvalidCredentials
	}
	return nil
}

// ValidatePassword checks password strength rules.
func ValidatePassword(p string) []string {
	var issues []string
	if len(p) < 8 {
		issues = append(issues, "must be at least 8 characters")
	}
	hasUpper := false
	hasDigit := false
	for _, r := range p {
		if r >= 'A' && r <= 'Z' {
			hasUpper = true
		}
		if r >= '0' && r <= '9' {
			hasDigit = true
		}
	}
	if !hasUpper {
		issues = append(issues, "must contain an uppercase letter")
	}
	if !hasDigit {
		issues = append(issues, "must contain a digit")
	}
	if !strings.ContainsAny(p, "!@#$%^&*()-_=+[]{}|;:',.<>?/") {
		issues = append(issues, "must contain a special character")
	}
	return issues
}
