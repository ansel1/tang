package users

import "testing"

func TestUsers(t *testing.T) {
	t.Skip("skipped test")
}

func TestSubtests(t *testing.T) {
	t.Run("subtest1", func(t *testing.T) {
		t.Skip("skipped subtest")
	})
	t.Run("subtest2", func(t *testing.T) {
		// passes
	})
	t.Run("subtest3", func(t *testing.T) {
		t.Run("subsubtest", func(t *testing.T) {
			// passes
		})
	})
	t.Run("subtest4", func(t *testing.T) {
		t.Fail()
	})
}
