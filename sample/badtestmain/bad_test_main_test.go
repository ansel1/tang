package badtestmain

import "testing"

func TestMain(m *testing.M) {
	panic("test main paniced")
}

func TestSplines(t *testing.T) {
	t.Log("reticulated splines")
}
