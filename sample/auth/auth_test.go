package auth

import (
	"errors"
	"testing"
	"time"
)

func TestRegister(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		s := NewStore()
		if err := s.Register("alice", "secureP4ss!"); err != nil {
			t.Fatalf("Register() unexpected error: %v", err)
		}
	})

	t.Run("weak_password", func(t *testing.T) {
		t.Parallel()
		s := NewStore()
		err := s.Register("bob", "short")
		if !errors.Is(err, ErrWeakPassword) {
			t.Errorf("Register() error = %v, want ErrWeakPassword", err)
		}
	})
}

func TestAuthenticate(t *testing.T) {
	t.Parallel()
	s := NewStore()
	_ = s.Register("alice", "correctPassword1!")

	t.Run("valid_credentials", func(t *testing.T) {
		t.Parallel()
		if err := s.Authenticate("alice", "correctPassword1!"); err != nil {
			t.Errorf("Authenticate() unexpected error: %v", err)
		}
	})

	t.Run("wrong_password", func(t *testing.T) {
		t.Parallel()
		err := s.Authenticate("alice", "wrongpass")
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Errorf("Authenticate() error = %v, want ErrInvalidCredentials", err)
		}
	})

	t.Run("unknown_user", func(t *testing.T) {
		t.Parallel()
		err := s.Authenticate("nobody", "password")
		if !errors.Is(err, ErrUserNotFound) {
			t.Errorf("Authenticate() error = %v, want ErrUserNotFound", err)
		}
	})

	t.Run("locked_account", func(t *testing.T) {
		t.Parallel()
		s := NewStore()
		_ = s.Register("locked", "validPassword1!")
		u := s.users["locked"]
		u.Locked = true
		s.users["locked"] = u

		err := s.Authenticate("locked", "validPassword1!")
		if !errors.Is(err, ErrAccountLocked) {
			t.Errorf("Authenticate() error = %v, want ErrAccountLocked", err)
		}
	})
}

func TestValidatePassword(t *testing.T) {
	t.Parallel()

	t.Run("strong_password", func(t *testing.T) {
		t.Parallel()
		issues := ValidatePassword("Str0ng!Pass")
		if len(issues) != 0 {
			t.Errorf("ValidatePassword() found issues for strong password: %v", issues)
		}
	})

	t.Run("too_short", func(t *testing.T) {
		t.Parallel()
		issues := ValidatePassword("Ab1!")
		found := false
		for _, issue := range issues {
			if issue == "must be at least 8 characters" {
				found = true
			}
		}
		if !found {
			t.Error("ValidatePassword() should report 'too short' for 4-char password")
		}
	})

	t.Run("missing_uppercase", func(t *testing.T) {
		t.Parallel()
		issues := ValidatePassword("lowercase1!")
		found := false
		for _, issue := range issues {
			if issue == "must contain an uppercase letter" {
				found = true
			}
		}
		if !found {
			t.Error("ValidatePassword() should report missing uppercase")
		}
	})

	t.Run("missing_digit", func(t *testing.T) {
		t.Parallel()
		issues := ValidatePassword("NoDigits!Here")
		found := false
		for _, issue := range issues {
			if issue == "must contain a digit" {
				found = true
			}
		}
		if !found {
			t.Error("ValidatePassword() should report missing digit")
		}
	})

	// Deliberately wrong assertion to produce a test failure.
	t.Run("missing_special_FAILS", func(t *testing.T) {
		t.Parallel()
		issues := ValidatePassword("NoSpecial1Here")
		t.Logf("Issues found: %v", issues)
		// BUG: this asserts the wrong count to demonstrate a failure in tang.
		if len(issues) != 0 {
			t.Errorf("expected 0 issues but got %d: %v", len(issues), issues)
		}
	})
}

// TestPasswordHashingNotImplemented is skipped because the feature doesn't exist yet.
func TestPasswordHashingNotImplemented(t *testing.T) {
	t.Skip("password hashing not yet implemented (tracked in #42)")
}

// TestRateLimiting simulates rate-limit attempts with visible timing.
func TestRateLimiting(t *testing.T) {
	t.Log("Simulating rate limit window...")
	s := NewStore()
	_ = s.Register("user", "password123!")

	for i := range 8 {
		_ = s.Authenticate("user", "wrong")
		t.Logf("  attempt %d/8: failed auth (expected)", i+1)
		time.Sleep(400 * time.Millisecond)
	}
	t.Log("Rate limit window complete, verifying lockout...")
	time.Sleep(500 * time.Millisecond)
	t.Log("Lockout verification done")
}

// TestBruteForceProtection simulates a longer brute-force scenario
// with parallel subtests that each take ~4 seconds.
func TestBruteForceProtection(t *testing.T) {
	t.Parallel()
	t.Log("Starting brute force protection tests...")

	t.Run("dictionary_attack", func(t *testing.T) {
		t.Parallel()
		s := NewStore()
		_ = s.Register("target", "correctHorse1!")

		passwords := []string{
			"password", "123456", "qwerty", "letmein",
			"admin", "welcome", "monkey", "dragon",
		}
		for i, pw := range passwords {
			time.Sleep(500 * time.Millisecond)
			err := s.Authenticate("target", pw)
			t.Logf("  dictionary attempt %d/%d: %q → %v", i+1, len(passwords), pw, err)
		}
		t.Log("  dictionary attack simulation complete")
	})

	t.Run("credential_stuffing", func(t *testing.T) {
		t.Parallel()
		s := NewStore()
		_ = s.Register("alice", "alicePass1!")
		_ = s.Register("bob", "bobSecure2@")

		creds := []struct{ user, pass string }{
			{"alice", "leaked1"}, {"bob", "leaked2"},
			{"alice", "leaked3"}, {"bob", "leaked4"},
			{"alice", "leaked5"}, {"bob", "leaked6"},
			{"alice", "alicePass1!"}, // correct one
		}
		for i, c := range creds {
			time.Sleep(500 * time.Millisecond)
			err := s.Authenticate(c.user, c.pass)
			status := "blocked"
			if err == nil {
				status = "SUCCESS"
			}
			t.Logf("  stuffing attempt %d/%d: %s/%s → %s", i+1, len(creds), c.user, c.pass, status)
		}
		t.Log("  credential stuffing simulation complete")
	})

	t.Run("timing_analysis", func(t *testing.T) {
		t.Parallel()
		s := NewStore()
		_ = s.Register("user", "password1!")

		t.Log("  measuring auth timing for valid vs invalid users...")
		for i := range 6 {
			time.Sleep(600 * time.Millisecond)
			start := time.Now()
			if i%2 == 0 {
				_ = s.Authenticate("user", "wrong")
			} else {
				_ = s.Authenticate("nonexistent", "wrong")
			}
			elapsed := time.Since(start)
			t.Logf("  timing check %d/6: %v", i+1, elapsed)
		}
		t.Log("  timing analysis complete")
	})
}
