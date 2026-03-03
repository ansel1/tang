package validator

import (
	"testing"
	"time"
)

func TestRequired(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		if err := Required("hello"); err != nil {
			t.Errorf("Required() unexpected error: %v", err)
		}
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		if err := Required(""); err == nil {
			t.Error("Required() should fail on empty string")
		}
	})

	t.Run("whitespace_only", func(t *testing.T) {
		t.Parallel()
		if err := Required("   "); err == nil {
			t.Error("Required() should fail on whitespace-only string")
		}
	})
}

func TestMinLength(t *testing.T) {
	t.Parallel()
	rule := MinLength(5)

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		if err := rule("hello"); err != nil {
			t.Errorf("MinLength(5) unexpected error for %q: %v", "hello", err)
		}
	})

	t.Run("too_short", func(t *testing.T) {
		t.Parallel()
		if err := rule("hi"); err == nil {
			t.Error("MinLength(5) should fail for 2-char string")
		}
	})

	t.Run("exact", func(t *testing.T) {
		t.Parallel()
		if err := rule("12345"); err != nil {
			t.Errorf("MinLength(5) unexpected error for exact-length string: %v", err)
		}
	})
}

func TestMaxLength(t *testing.T) {
	t.Parallel()
	rule := MaxLength(5)

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		if err := rule("hi"); err != nil {
			t.Errorf("MaxLength(5) unexpected error: %v", err)
		}
	})

	t.Run("too_long", func(t *testing.T) {
		t.Parallel()
		if err := rule("way too long"); err == nil {
			t.Error("MaxLength(5) should fail for long string")
		}
	})
}

func TestMatchesPattern(t *testing.T) {
	t.Parallel()

	t.Run("email", func(t *testing.T) {
		t.Parallel()
		rule := MatchesPattern(`^[^@]+@[^@]+\.[^@]+$`, "email format")

		t.Run("valid", func(t *testing.T) {
			t.Parallel()
			if err := rule("user@example.com"); err != nil {
				t.Errorf("email pattern unexpected error: %v", err)
			}
		})

		t.Run("invalid", func(t *testing.T) {
			t.Parallel()
			if err := rule("not-an-email"); err == nil {
				t.Error("email pattern should reject invalid email")
			}
		})

		t.Run("missing_domain", func(t *testing.T) {
			t.Parallel()
			if err := rule("user@"); err == nil {
				t.Error("email pattern should reject missing domain")
			}
		})
	})

	t.Run("phone", func(t *testing.T) {
		t.Parallel()
		rule := MatchesPattern(`^\+?[0-9]{10,15}$`, "phone number")

		t.Run("valid_US", func(t *testing.T) {
			t.Parallel()
			if err := rule("+12025551234"); err != nil {
				t.Errorf("phone pattern unexpected error: %v", err)
			}
		})

		t.Run("too_short", func(t *testing.T) {
			t.Parallel()
			if err := rule("12345"); err == nil {
				t.Error("phone pattern should reject short number")
			}
		})

		t.Run("letters", func(t *testing.T) {
			t.Parallel()
			if err := rule("123-abc-4567"); err == nil {
				t.Error("phone pattern should reject letters")
			}
		})
	})

	t.Run("url", func(t *testing.T) {
		t.Parallel()
		rule := MatchesPattern(`^https?://[^\s]+$`, "URL format")

		t.Run("valid_https", func(t *testing.T) {
			t.Parallel()
			if err := rule("https://example.com"); err != nil {
				t.Errorf("url pattern unexpected error: %v", err)
			}
		})

		t.Run("valid_http", func(t *testing.T) {
			t.Parallel()
			if err := rule("http://localhost:8080/path"); err != nil {
				t.Errorf("url pattern unexpected error: %v", err)
			}
		})

		t.Run("missing_scheme", func(t *testing.T) {
			t.Parallel()
			if err := rule("example.com"); err == nil {
				t.Error("url pattern should reject missing scheme")
			}
		})
	})
}

func TestValidate(t *testing.T) {
	t.Parallel()

	t.Run("all_pass", func(t *testing.T) {
		t.Parallel()
		errs := Validate("hello", Required, MinLength(3), MaxLength(10))
		if len(errs) != 0 {
			t.Errorf("Validate() returned %d errors, want 0: %v", len(errs), errs)
		}
	})

	t.Run("multiple_failures", func(t *testing.T) {
		t.Parallel()
		errs := Validate("", Required, MinLength(5))
		if len(errs) != 2 {
			t.Errorf("Validate() returned %d errors, want 2: %v", len(errs), errs)
		}
		t.Logf("Validation errors (expected): %v", errs)
	})

	t.Run("partial_failure", func(t *testing.T) {
		t.Parallel()
		errs := Validate("hi", Required, MinLength(5), MaxLength(10))
		if len(errs) != 1 {
			t.Errorf("Validate() returned %d errors, want 1: %v", len(errs), errs)
		}
	})
}

// TestSlowValidation simulates a slow validation pipeline with
// periodic logging. This is the longest-running test in the sample.
func TestSlowValidation(t *testing.T) {
	t.Log("Running validation pipeline with simulated I/O delays...")

	emailRule := MatchesPattern(`^[^@]+@[^@]+\.[^@]+$`, "email")
	inputs := []string{
		"alice@example.com",
		"bob",
		"",
		"valid@test.org",
		"x",
		"admin@internal.corp",
		"not-valid",
		"user123@domain.io",
		"test",
		"support@company.com",
	}

	for i, input := range inputs {
		time.Sleep(1 * time.Second)
		errs := Validate(input, Required, emailRule, MinLength(3))
		status := "PASS"
		if len(errs) > 0 {
			status = "FAIL"
		}
		t.Logf("  [%d/%d] validate(%q): %s (%d errors)", i+1, len(inputs), input, status, len(errs))
	}
	t.Log("Validation pipeline complete")
}

// TestFormValidation exercises nested parallel subtests simulating
// form field validation, each taking 3-4 seconds.
func TestFormValidation(t *testing.T) {
	t.Parallel()
	t.Log("Validating form submissions...")

	t.Run("user_registration", func(t *testing.T) {
		t.Parallel()
		fields := map[string]struct {
			value string
			rules []Rule
		}{
			"username": {"alice", []Rule{Required, MinLength(3), MaxLength(20)}},
			"email":    {"alice@example.com", []Rule{Required, MatchesPattern(`^[^@]+@[^@]+\.[^@]+$`, "email")}},
			"password": {"SecureP4ss!", []Rule{Required, MinLength(8)}},
			"name":     {"Alice Smith", []Rule{Required, MinLength(2), MaxLength(50)}},
			"bio":      {"Go developer", []Rule{MaxLength(200)}},
		}
		for field, spec := range fields {
			time.Sleep(1 * time.Second)
			errs := Validate(spec.value, spec.rules...)
			t.Logf("  %s = %q → %d errors", field, spec.value, len(errs))
			if len(errs) != 0 {
				t.Errorf("field %q should pass: %v", field, errs)
			}
		}
		t.Log("  user registration form: all fields valid")
	})

	t.Run("contact_form", func(t *testing.T) {
		t.Parallel()
		type field struct {
			name       string
			value      string
			rules      []Rule
			wantErrors int
		}
		fields := []field{
			{"name", "", []Rule{Required, MinLength(2)}, 2},
			{"email", "not-email", []Rule{Required, MatchesPattern(`^[^@]+@[^@]+\.[^@]+$`, "email")}, 1},
			{"subject", "Hi", []Rule{Required, MinLength(5)}, 1},
			{"message", "Help me with my account please", []Rule{Required, MinLength(10), MaxLength(1000)}, 0},
			{"phone", "+12025551234", []Rule{Required, MatchesPattern(`^\+?[0-9]{10,15}$`, "phone")}, 0},
		}
		for _, f := range fields {
			time.Sleep(1 * time.Second)
			errs := Validate(f.value, f.rules...)
			t.Logf("  %s = %q → %d errors (expected %d)", f.name, f.value, len(errs), f.wantErrors)
			if len(errs) != f.wantErrors {
				t.Errorf("field %q: got %d errors, want %d: %v", f.name, len(errs), f.wantErrors, errs)
			}
		}
		t.Log("  contact form: validation results match expectations")
	})

	t.Run("settings_form", func(t *testing.T) {
		t.Parallel()
		type setting struct {
			key   string
			value string
			rules []Rule
		}
		settings := []setting{
			{"display_name", "Alice", []Rule{Required, MinLength(1), MaxLength(30)}},
			{"timezone", "America/New_York", []Rule{Required}},
			{"language", "en", []Rule{Required, MinLength(2), MaxLength(5)}},
			{"theme", "dark", []Rule{Required}},
			{"page_size", "25", []Rule{Required}},
			{"notification_email", "alice@example.com", []Rule{Required, MatchesPattern(`^[^@]+@[^@]+\.[^@]+$`, "email")}},
		}
		for _, s := range settings {
			time.Sleep(1 * time.Second)
			errs := Validate(s.value, s.rules...)
			t.Logf("  %s = %q → %d errors", s.key, s.value, len(errs))
			if len(errs) != 0 {
				t.Errorf("setting %q should pass: %v", s.key, errs)
			}
		}
		t.Log("  settings form: all settings valid")
	})
}

// TestSkippedDatabaseValidation is skipped.
func TestSkippedDatabaseValidation(t *testing.T) {
	t.Skip("requires database connection")
}
