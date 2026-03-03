package mathutil

import "math"

// Add returns the sum of two integers.
func Add(a, b int) int { return a + b }

// Subtract returns a minus b.
func Subtract(a, b int) int { return a - b }

// Multiply returns the product of two integers.
func Multiply(a, b int) int { return a * b }

// Divide returns a divided by b. Panics on division by zero.
func Divide(a, b int) int {
	if b == 0 {
		panic("division by zero")
	}
	return a / b
}

// Fibonacci returns the nth Fibonacci number.
func Fibonacci(n int) int {
	if n <= 1 {
		return n
	}
	a, b := 0, 1
	for i := 2; i <= n; i++ {
		a, b = b, a+b
	}
	return b
}

// IsPrime reports whether n is a prime number.
func IsPrime(n int) bool {
	if n < 2 {
		return false
	}
	for i := 2; i <= int(math.Sqrt(float64(n))); i++ {
		if n%i == 0 {
			return false
		}
	}
	return true
}
