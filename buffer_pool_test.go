package main

import (
	"testing"
)

func TestBufferPoolIsCorrectlyInitialized(t *testing.T) {
	const (
		B_NUMBER = 10
		B_CAP    = 124
	)
	bp := NewBufferPool(B_NUMBER, B_CAP)

	b := bp.Get()
	defer bp.Put(b)

	if b.Cap() != B_CAP {
		t.Errorf("Invalid buffer capacity: %d\n", b.Cap())
	}
}
