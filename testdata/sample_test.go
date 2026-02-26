package testdata

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestPass(t *testing.T) {
	t.Log("This test passes")
}

func TestFail(t *testing.T) {
	t.Log("This test fails")
	t.Fail()
}

func TestSkip(t *testing.T) {
	t.Skip("This test is skipped")
}

func TestSlow(t *testing.T) {
	t.Log("This test takes a moment")
	time.Sleep(100 * time.Millisecond)
}

func TestAnother(t *testing.T) {
	t.Log("Another passing test")
}

func TestMain(m *testing.M) {
	fmt.Fprintln(os.Stderr, "This is output from TestMain")
	m.Run()
}

func TestOutput(t *testing.T) {
	fmt.Fprintln(os.Stderr, "This is output from a test")
}
