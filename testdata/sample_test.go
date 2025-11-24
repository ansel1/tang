package testdata

import (
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
