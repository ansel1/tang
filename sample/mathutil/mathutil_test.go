package mathutil

import (
	"fmt"
	"testing"
	"time"
)

func TestAdd(t *testing.T) {
	t.Parallel()
	if got := Add(2, 3); got != 5 {
		t.Errorf("Add(2, 3) = %d, want 5", got)
	}
}

func TestSubtract(t *testing.T) {
	t.Parallel()
	if got := Subtract(10, 4); got != 6 {
		t.Errorf("Subtract(10, 4) = %d, want 6", got)
	}
}

func TestMultiply(t *testing.T) {
	t.Parallel()
	if got := Multiply(3, 7); got != 21 {
		t.Errorf("Multiply(3, 7) = %d, want 21", got)
	}
}

func TestDivide(t *testing.T) {
	t.Parallel()

	t.Run("basic", func(t *testing.T) {
		t.Parallel()
		if got := Divide(10, 2); got != 5 {
			t.Errorf("Divide(10, 2) = %d, want 5", got)
		}
	})

	t.Run("integer_truncation", func(t *testing.T) {
		t.Parallel()
		if got := Divide(7, 2); got != 3 {
			t.Errorf("Divide(7, 2) = %d, want 3", got)
		}
	})

	t.Run("negative", func(t *testing.T) {
		t.Parallel()
		if got := Divide(-10, 2); got != -5 {
			t.Errorf("Divide(-10, 2) = %d, want -5", got)
		}
	})
}

func TestFibonacci(t *testing.T) {
	tests := []struct {
		name string
		n    int
		want int
	}{
		{"zero", 0, 0},
		{"one", 1, 1},
		{"small", 5, 5},
		{"medium", 10, 55},
		{"larger", 20, 6765},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := Fibonacci(tc.n); got != tc.want {
				t.Errorf("Fibonacci(%d) = %d, want %d", tc.n, got, tc.want)
			}
		})
	}
}

// TestIsPrimeSlow runs prime checks with simulated work, producing
// parallel subtests that each take 3-4 seconds.
func TestIsPrimeSlow(t *testing.T) {
	t.Parallel()
	t.Log("Starting slow prime checks...")

	t.Run("small_primes", func(t *testing.T) {
		t.Parallel()
		primes := []int{2, 3, 5, 7, 11, 13}
		for i, p := range primes {
			t.Logf("  checking %d...", p)
			time.Sleep(500 * time.Millisecond)
			if !IsPrime(p) {
				t.Errorf("IsPrime(%d) = false, want true", p)
			}
			_ = i
		}
		t.Log("  small primes complete")
	})

	t.Run("large_primes", func(t *testing.T) {
		t.Parallel()
		primes := []int{97, 101, 103, 107, 109, 113}
		for _, p := range primes {
			t.Logf("  verifying %d is prime...", p)
			time.Sleep(600 * time.Millisecond)
			if !IsPrime(p) {
				t.Errorf("IsPrime(%d) = false, want true", p)
			}
		}
		t.Log("  large primes complete")
	})

	t.Run("composites", func(t *testing.T) {
		t.Parallel()
		composites := []int{4, 6, 8, 9, 10, 12, 15}
		for _, c := range composites {
			t.Logf("  verifying %d is composite...", c)
			time.Sleep(400 * time.Millisecond)
			if IsPrime(c) {
				t.Errorf("IsPrime(%d) = true, want false", c)
			}
		}
		t.Log("  composites complete")
	})
}

// TestFibonacciSequential verifies that successive Fibonacci numbers satisfy
// the recurrence relation.  Each subtest depends on results captured by the
// previous one, so they must run sequentially (no t.Parallel).
func TestFibonacciSequential(t *testing.T) {
	var prev, cur int

	t.Run("fib(0)", func(t *testing.T) {
		prev = Fibonacci(0)
		if prev != 0 {
			t.Fatalf("Fibonacci(0) = %d, want 0", prev)
		}
		t.Logf("Fibonacci(0) = %d", prev)
	})

	t.Run("fib(1)", func(t *testing.T) {
		cur = Fibonacci(1)
		if cur != 1 {
			t.Fatalf("Fibonacci(1) = %d, want 1", cur)
		}
		t.Logf("Fibonacci(1) = %d", cur)
	})

	for i := 2; i <= 10; i++ {
		t.Run(fmt.Sprintf("fib(%d)", i), func(t *testing.T) {
			next := Fibonacci(i)
			want := prev + cur
			if next != want {
				t.Fatalf("Fibonacci(%d) = %d, want %d (= %d + %d)", i, next, want, prev, cur)
			}
			t.Logf("Fibonacci(%d) = %d = %d + %d ✓", i, next, prev, cur)
			prev, cur = cur, next
		})
	}
}

func TestIsPrime(t *testing.T) {
	tests := []struct {
		name string
		n    int
		want bool
	}{
		{"negative", -1, false},
		{"zero", 0, false},
		{"one", 1, false},
		{"two", 2, true},
		{"three", 3, true},
		{"four", 4, false},
		{"large_prime", 97, true},
		{"large_composite", 100, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := IsPrime(tc.n); got != tc.want {
				t.Errorf("IsPrime(%d) = %v, want %v", tc.n, got, tc.want)
			}
		})
	}
}
