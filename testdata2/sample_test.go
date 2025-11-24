package testdata2

import (
	"testing"
	"time"
)

func TestPass1(t *testing.T) {
	t.Log("This test passes")
}

func TestPass2(t *testing.T) {
	t.Log("Another passing test")
}

func TestSlow(t *testing.T) {
	t.Log("This test takes a moment")
	time.Sleep(100 * time.Millisecond)
}

func TestSubtest(t *testing.T) {
	t.Run("Subtest1", func(t *testing.T) {
		t.Log("This is subtest 1")
		time.Sleep(500 * time.Millisecond)
	})

	t.Run("Subtest2", func(t *testing.T) {
		t.Log("This is subtest 2")
		time.Sleep(500 * time.Millisecond)
	})

	t.Run("SubtestWithSubtest", func(t *testing.T) {
		t.Log("This is subtest has a subtest")
		t.Run("Subsubtest", func(t *testing.T) {
			t.Log("This is a subsubtest")
			time.Sleep(500 * time.Millisecond)
		})
		t.Run("failing subtest", func(t *testing.T) {
			t.Log("This is a failing subtest")
			time.Sleep(500 * time.Millisecond)
			t.Fail()
		})

		t.Run("Skipped subtest", func(t *testing.T) {
			t.Skip("skipping subsubtest")
		})
	})

}
