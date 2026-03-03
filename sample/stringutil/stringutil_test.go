package stringutil

import (
	"fmt"
	"testing"
	"time"
)

func TestReverse(t *testing.T) {
	t.Parallel()

	t.Run("simple", func(t *testing.T) {
		t.Parallel()
		got := Reverse("hello")
		if got != "olleh" {
			t.Errorf("Reverse(%q) = %q, want %q", "hello", got, "olleh")
		}
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		got := Reverse("")
		if got != "" {
			t.Errorf("Reverse(%q) = %q, want %q", "", got, "")
		}
	})

	t.Run("unicode", func(t *testing.T) {
		t.Parallel()
		got := Reverse("日本語")
		if got != "語本日" {
			t.Errorf("Reverse(%q) = %q, want %q", "日本語", got, "語本日")
		}
	})

	t.Run("single_char", func(t *testing.T) {
		t.Parallel()
		got := Reverse("x")
		if got != "x" {
			t.Errorf("Reverse(%q) = %q, want %q", "x", got, "x")
		}
	})
}

// TestPalindromeStress runs palindrome checks with simulated work.
// The parallel subtests each take 3-4 seconds.
func TestPalindromeStress(t *testing.T) {
	t.Parallel()
	t.Log("Running palindrome stress tests with simulated I/O...")

	t.Run("known_palindromes", func(t *testing.T) {
		t.Parallel()
		words := []string{"racecar", "madam", "level", "rotor", "civic", "kayak", "refer"}
		for _, w := range words {
			time.Sleep(500 * time.Millisecond)
			t.Logf("  checking %q...", w)
			if !Palindrome(w) {
				t.Errorf("Palindrome(%q) = false, want true", w)
			}
		}
		t.Log("  palindromes verified")
	})

	t.Run("non_palindromes", func(t *testing.T) {
		t.Parallel()
		words := []string{"hello", "world", "golang", "testing", "sample", "runner"}
		for _, w := range words {
			time.Sleep(500 * time.Millisecond)
			t.Logf("  checking %q...", w)
			if Palindrome(w) {
				t.Errorf("Palindrome(%q) = true, want false", w)
			}
		}
		t.Log("  non-palindromes verified")
	})
}

func TestPalindrome(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		s    string
		want bool
	}{
		{"racecar", "racecar", true},
		{"hello", "hello", false},
		{"mixed_case", "Racecar", true},
		{"single_char", "a", true},
		{"empty", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := Palindrome(tc.s); got != tc.want {
				t.Errorf("Palindrome(%q) = %v, want %v", tc.s, got, tc.want)
			}
		})
	}
}

func TestWordCount(t *testing.T) {
	t.Parallel()
	t.Log("Testing word count with various inputs")

	cases := []struct {
		input string
		want  int
	}{
		{"hello world", 2},
		{"", 0},
		{"one", 1},
		{"  lots   of   spaces  ", 3},
	}
	for _, c := range cases {
		if got := WordCount(c.input); got != c.want {
			t.Errorf("WordCount(%q) = %d, want %d", c.input, got, c.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	t.Parallel()

	t.Run("no_truncation", func(t *testing.T) {
		t.Parallel()
		got := Truncate("hi", 10)
		if got != "hi" {
			t.Errorf("got %q, want %q", got, "hi")
		}
	})

	t.Run("exact_length", func(t *testing.T) {
		t.Parallel()
		got := Truncate("hello", 5)
		if got != "hello" {
			t.Errorf("got %q, want %q", got, "hello")
		}
	})

	t.Run("truncated", func(t *testing.T) {
		t.Parallel()
		got := Truncate("hello world", 8)
		if got != "hello..." {
			t.Errorf("got %q, want %q", got, "hello...")
		}
	})

	t.Run("very_short_max", func(t *testing.T) {
		t.Parallel()
		got := Truncate("hello", 2)
		if got != "he" {
			t.Errorf("got %q, want %q", got, "he")
		}
	})
}

// TestSlowReverse is deliberately slow, with periodic log output to
// exercise tang's output capture and elapsed time display.
func TestSlowReverse(t *testing.T) {
	words := []string{"apple", "banana", "cherry", "dragonfruit", "elderberry", "fig", "guava"}
	for i, w := range words {
		t.Logf("  reversing word %d/%d: %q...", i+1, len(words), w)
		time.Sleep(800 * time.Millisecond)
		got := Reverse(w)
		expected := Reverse(got) // reverse twice = original
		if expected != w {
			t.Errorf("double-reverse of %q gave %q", w, expected)
		}
	}
	t.Log("All reversals verified")
}

// TestReverseBatch runs a large batch of reverse operations with
// parallel subtests that each take ~3 seconds.
func TestReverseBatch(t *testing.T) {
	t.Parallel()
	t.Log("Starting batch reverse tests...")

	t.Run("short_words", func(t *testing.T) {
		t.Parallel()
		words := []string{"go", "is", "fun", "and", "fast", "cool"}
		for _, w := range words {
			time.Sleep(500 * time.Millisecond)
			got := Reverse(Reverse(w))
			if got != w {
				t.Errorf("double reverse of %q = %q", w, got)
			}
			t.Logf("  verified %q", w)
		}
	})

	t.Run("long_words", func(t *testing.T) {
		t.Parallel()
		words := []string{"extraordinary", "consciousness", "Mediterranean", "approximately", "communication"}
		for _, w := range words {
			time.Sleep(700 * time.Millisecond)
			got := Reverse(Reverse(w))
			if got != w {
				t.Errorf("double reverse of %q = %q", w, got)
			}
			t.Logf("  verified %q (%d chars)", w, len(w))
		}
	})

	t.Run("sentences", func(t *testing.T) {
		t.Parallel()
		sentences := []string{
			"the quick brown fox",
			"jumps over the lazy dog",
			"hello world from go",
			"testing is important",
		}
		for _, s := range sentences {
			time.Sleep(800 * time.Millisecond)
			reversed := Reverse(s)
			t.Logf("  %q -> %q", s, reversed)
			back := Reverse(reversed)
			if back != s {
				t.Errorf("round-trip failed for %q", s)
			}
		}
	})
}

// TestSkippedFeature is skipped with a reason message.
func TestSkippedFeature(t *testing.T) {
	t.Skip("Skipping: regex support not yet implemented")
}

// TestConditionalSkip demonstrates conditional skipping.
func TestConditionalSkip(t *testing.T) {
	t.Skip("Skipping: requires external service")
}

// TestWordCountSlow demonstrates a slow sequential test with output.
func TestWordCountSlow(t *testing.T) {
	t.Log("Counting words in sample paragraphs...")
	paragraphs := []string{
		"The quick brown fox jumps over the lazy dog",
		"Go is an open source programming language",
		"Testing is a crucial part of software development",
		"Tang provides a better test runner experience",
		"Concurrency is not parallelism as Rob Pike says",
		"Simplicity is the ultimate sophistication",
	}
	for i, p := range paragraphs {
		time.Sleep(700 * time.Millisecond)
		count := WordCount(p)
		t.Logf("  paragraph %d: %d words — %q", i+1, count, fmt.Sprintf("%.30s...", p))
	}
}
