package parallelcrash

import (
	"fmt"
	"testing"
	"time"
)

func ptr(s string) *string { return &s }

func TestTransformParallel(t *testing.T) {
	t.Parallel()

	t.Run("basic", func(t *testing.T) {
		t.Parallel()
		t.Log("transforming 3 items...")
		time.Sleep(3 * time.Second)
		got := Transform([]*string{ptr("a"), ptr("b"), ptr("c")})
		if len(got) != 3 {
			t.Errorf("got %d results, want 3", len(got))
		}
		t.Log("basic transform OK")
	})

	t.Run("single", func(t *testing.T) {
		t.Parallel()
		t.Log("transforming single item...")
		time.Sleep(4 * time.Second)
		got := Transform([]*string{ptr("only")})
		if got[0] != "transformed:only" {
			t.Errorf("got %q, want %q", got[0], "transformed:only")
		}
		t.Log("single transform OK")
	})

	t.Run("concurrent_map_writes", func(t *testing.T) {
		t.Parallel()
		t.Log("setting up concurrent map writers...")
		time.Sleep(500 * time.Millisecond)
		t.Log("triggering concurrent map writes (runtime fatal)...")
		ConcurrentMapWrites()
		t.Log("this line is never reached")
	})

	t.Run("large_batch", func(t *testing.T) {
		t.Parallel()
		t.Log("building large batch...")
		items := make([]*string, 100)
		for i := range items {
			items[i] = ptr(fmt.Sprintf("item-%d", i))
		}
		time.Sleep(3500 * time.Millisecond)
		got := Transform(items)
		if len(got) != 100 {
			t.Errorf("got %d results, want 100", len(got))
		}
		t.Log("large batch OK")
	})

	t.Run("empty_input", func(t *testing.T) {
		t.Parallel()
		t.Log("transforming empty slice...")
		time.Sleep(3 * time.Second)
		got := Transform([]*string{})
		if len(got) != 0 {
			t.Errorf("got %d results, want 0", len(got))
		}
		t.Log("empty input OK")
	})
}

func TestMergeParallel(t *testing.T) {
	t.Parallel()

	t.Run("equal_lengths", func(t *testing.T) {
		t.Parallel()
		t.Log("merging equal-length slices...")
		time.Sleep(3 * time.Second)
		got := Merge([]string{"a", "b"}, []string{"1", "2"})
		if got[0] != "a+1" {
			t.Errorf("got %q, want %q", got[0], "a+1")
		}
		t.Log("equal lengths OK")
	})

	t.Run("single_element", func(t *testing.T) {
		t.Parallel()
		t.Log("merging single-element slices...")
		time.Sleep(3500 * time.Millisecond)
		got := Merge([]string{"x"}, []string{"y"})
		if got[0] != "x+y" {
			t.Errorf("got %q, want %q", got[0], "x+y")
		}
		t.Log("single element OK")
	})
}
