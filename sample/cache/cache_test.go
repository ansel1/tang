package cache

import (
	"fmt"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	t.Parallel()
	c := New(time.Second)
	if c == nil {
		t.Fatal("New returned nil")
	}
	if c.Len() != 0 {
		t.Errorf("new cache Len() = %d, want 0", c.Len())
	}
}

func TestSetAndGet(t *testing.T) {
	t.Parallel()
	c := New(time.Minute)

	t.Run("store_and_retrieve", func(t *testing.T) {
		t.Parallel()
		c := New(time.Minute)
		c.Set("key1", "value1")
		got, ok := c.Get("key1")
		if !ok || got != "value1" {
			t.Errorf("Get(key1) = (%q, %v), want (value1, true)", got, ok)
		}
	})

	t.Run("missing_key", func(t *testing.T) {
		t.Parallel()
		got, ok := c.Get("nonexistent")
		if ok {
			t.Errorf("Get(nonexistent) = (%q, true), want ('', false)", got)
		}
	})

	t.Run("overwrite", func(t *testing.T) {
		t.Parallel()
		c := New(time.Minute)
		c.Set("k", "v1")
		c.Set("k", "v2")
		got, _ := c.Get("k")
		if got != "v2" {
			t.Errorf("after overwrite, Get(k) = %q, want v2", got)
		}
	})
}

// TestExpiration exercises cache TTL with a longer wait to show
// tang's elapsed time ticking.
func TestExpiration(t *testing.T) {
	t.Log("Creating cache with 1.5s TTL")
	c := New(1500 * time.Millisecond)

	c.Set("key-a", "alpha")
	c.Set("key-b", "bravo")
	c.Set("key-c", "charlie")

	t.Log("Verifying all keys exist before expiration...")
	for _, k := range []string{"key-a", "key-b", "key-c"} {
		if _, ok := c.Get(k); !ok {
			t.Fatalf("key %q should exist before TTL", k)
		}
		t.Logf("  %s: present", k)
		time.Sleep(300 * time.Millisecond)
	}

	t.Log("Waiting for TTL to expire...")
	time.Sleep(1500 * time.Millisecond)

	t.Log("Verifying all keys have expired...")
	for _, k := range []string{"key-a", "key-b", "key-c"} {
		if _, ok := c.Get(k); ok {
			t.Errorf("key %q should have expired", k)
		}
		t.Logf("  %s: expired", k)
		time.Sleep(200 * time.Millisecond)
	}
	t.Log("Expiration test complete")
}

// TestPurge exercises the purge mechanism with visible delays.
func TestPurge(t *testing.T) {
	t.Log("Setting up cache for purge test")
	c := New(500 * time.Millisecond)

	for i := range 8 {
		c.Set(fmt.Sprintf("key%d", i), fmt.Sprintf("val%d", i))
		t.Logf("  stored key%d", i)
		time.Sleep(200 * time.Millisecond)
	}
	t.Logf("Cache has %d items", c.Len())

	t.Log("Waiting for oldest entries to expire...")
	time.Sleep(600 * time.Millisecond)

	removed := c.Purge()
	t.Logf("Purged %d expired entries, %d remaining", removed, c.Len())

	if c.Len()+removed != 8 {
		t.Errorf("purge accounting mismatch: removed=%d, remaining=%d, total should be 8", removed, c.Len())
	}
}

// TestConcurrentAccess verifies thread safety under concurrent load,
// with parallel subtests that each run for ~3 seconds.
func TestConcurrentAccess(t *testing.T) {
	t.Parallel()
	t.Log("Starting concurrent access tests...")

	t.Run("writers_only", func(t *testing.T) {
		t.Parallel()
		c := New(time.Minute)
		done := make(chan struct{})

		for i := range 5 {
			go func() {
				defer func() { done <- struct{}{} }()
				for j := range 50 {
					c.Set(fmt.Sprintf("w%d-k%d", i, j), "value")
					time.Sleep(10 * time.Millisecond)
				}
			}()
		}
		for range 5 {
			<-done
		}
		t.Logf("  writers complete, cache has %d entries", c.Len())
	})

	t.Run("readers_only", func(t *testing.T) {
		t.Parallel()
		c := New(time.Minute)
		// Pre-populate.
		for i := range 100 {
			c.Set(fmt.Sprintf("k%d", i), "v")
		}
		done := make(chan struct{})

		for i := range 5 {
			go func() {
				defer func() { done <- struct{}{} }()
				for j := range 100 {
					c.Get(fmt.Sprintf("k%d", (i*100+j)%100))
					time.Sleep(5 * time.Millisecond)
				}
			}()
		}
		for range 5 {
			<-done
		}
		t.Log("  readers complete")
	})

	t.Run("mixed_read_write", func(t *testing.T) {
		t.Parallel()
		c := New(time.Minute)
		done := make(chan struct{})

		// Writers.
		for i := range 5 {
			go func() {
				defer func() { done <- struct{}{} }()
				for j := range 50 {
					c.Set(fmt.Sprintf("m%d-k%d", i, j), "value")
					time.Sleep(10 * time.Millisecond)
				}
			}()
		}
		// Readers.
		for i := range 5 {
			go func() {
				defer func() { done <- struct{}{} }()
				for j := range 50 {
					c.Get(fmt.Sprintf("m%d-k%d", i, j))
					time.Sleep(10 * time.Millisecond)
				}
			}()
		}
		// Purgers.
		for range 2 {
			go func() {
				defer func() { done <- struct{}{} }()
				for range 10 {
					c.Purge()
					time.Sleep(50 * time.Millisecond)
				}
			}()
		}
		for range 12 {
			<-done
		}
		t.Logf("  mixed test complete, cache has %d entries", c.Len())
	})
}

// TestCacheWarming simulates a slow cache warming process with logging.
func TestCacheWarming(t *testing.T) {
	t.Log("Warming cache from simulated data source...")
	c := New(time.Minute)

	sources := []string{"users", "products", "orders", "sessions", "config", "permissions", "templates", "analytics"}
	for i, src := range sources {
		time.Sleep(700 * time.Millisecond)
		for j := range 10 {
			c.Set(fmt.Sprintf("%s:%d", src, j), fmt.Sprintf("data-%d", j))
		}
		t.Logf("  loaded %s (%d/%d sources, %d total entries)", src, i+1, len(sources), c.Len())
	}
	t.Logf("Cache warming complete: %d entries loaded", c.Len())
}
