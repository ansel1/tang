package validator

import (
	"fmt"
	"regexp"
	"strings"
)

// Rule is a validation function returning an error if the value is invalid.
type Rule func(value string) error

// Required checks that the value is non-empty.
func Required(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("value is required")
	}
	return nil
}

// MinLength returns a Rule checking minimum string length.
func MinLength(n int) Rule {
	return func(value string) error {
		if len(value) < n {
			return fmt.Errorf("must be at least %d characters, got %d", n, len(value))
		}
		return nil
	}
}

// MaxLength returns a Rule checking maximum string length.
func MaxLength(n int) Rule {
	return func(value string) error {
		if len(value) > n {
			return fmt.Errorf("must be at most %d characters, got %d", n, len(value))
		}
		return nil
	}
}

// MatchesPattern returns a Rule checking against a regex pattern.
func MatchesPattern(pattern, description string) Rule {
	re := regexp.MustCompile(pattern)
	return func(value string) error {
		if !re.MatchString(value) {
			return fmt.Errorf("must match %s", description)
		}
		return nil
	}
}

// Validate runs all rules against the value and returns all errors.
func Validate(value string, rules ...Rule) []error {
	var errs []error
	for _, r := range rules {
		if err := r(value); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}
