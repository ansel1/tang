package panicker

import (
	"testing"
	"time"
)

func TestProcessItemsValid(t *testing.T) {
	t.Log("processing valid items...")
	time.Sleep(200 * time.Millisecond)
	got := ProcessItems([]string{"a", "b", "c"})
	if len(got) != 3 {
		t.Errorf("ProcessItems returned %d items, want 3", len(got))
	}
	t.Log("valid items processed successfully")
}

func TestBuildIndexUnique(t *testing.T) {
	t.Log("building index with unique keys...")
	time.Sleep(200 * time.Millisecond)

	m := BuildIndex([]string{"x", "y", "z"}, []int{1, 2, 3})
	if m["y"] != 2 {
		t.Errorf("index[y] = %d, want 2", m["y"])
	}
	t.Log("unique keys OK")
}

func TestSafeRecursion(t *testing.T) {
	t.Log("testing safe recursion depth of 5")
	time.Sleep(200 * time.Millisecond)
	got := RecursiveDepth(5)
	if got != 5 {
		t.Errorf("RecursiveDepth(5) = %d, want 5", got)
	}
	t.Log("safe depth OK")
}

func TestProcessItemsPanic(t *testing.T) {
	t.Log("processing first batch of valid items...")
	time.Sleep(300 * time.Millisecond)
	_ = ProcessItems([]string{"ok", "fine"})
	t.Log("first batch OK, now processing batch with empty item...")
	time.Sleep(300 * time.Millisecond)
	ProcessItems([]string{"ok", "", "never reached"})
	t.Log("this line is never reached")
}

func TestBuildIndexDuplicateKeyPanic(t *testing.T) {
	t.Log("building index with unique keys first...")
	time.Sleep(200 * time.Millisecond)

	m := BuildIndex([]string{"x", "y", "z"}, []int{1, 2, 3})
	t.Logf("built index with %d entries", len(m))

	time.Sleep(200 * time.Millisecond)
	t.Log("now building index with duplicate keys...")
	BuildIndex([]string{"a", "b", "a"}, []int{1, 2, 3})
	t.Log("this line is never reached")
}

func TestRecursiveDepthPanic(t *testing.T) {
	t.Log("testing recursion with decreasing depth...")
	for i := 3; i >= 0; i-- {
		time.Sleep(300 * time.Millisecond)
		t.Logf("calling RecursiveDepth(%d)", i)
		RecursiveDepth(i)
	}
	t.Log("this line is never reached")
}

func TestLengthMismatchPanic(t *testing.T) {
	t.Log("starting length mismatch test")
	time.Sleep(300 * time.Millisecond)

	t.Log("building index with matched lengths first...")
	m := BuildIndex([]string{"one"}, []int{1})
	t.Logf("got index with %d entries", len(m))

	time.Sleep(200 * time.Millisecond)
	t.Log("now calling with mismatched lengths...")
	BuildIndex([]string{"a", "b"}, []int{1})
	t.Log("this line is never reached")
}
