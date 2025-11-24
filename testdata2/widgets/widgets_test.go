package widgets

import (
	"testing"
	"time"
)

func TestPass(t *testing.T) {
	t.Log("passes")
}

func TestTimer(t *testing.T) {
	for i := 0; i < 10; i++ {
		time.Sleep(500 * time.Millisecond)
		t.Logf("Elapsed time: %v seconds", float64(i+1)*0.5)
	}
}
