package store

import (
	"testing"
)

func TestMemStoreLRU(t *testing.T) {
	st, err := NewMemStore(10)
	if err != nil {
		t.Fatal(err)
	}
	storeTest(t, st)
}

func TestMemStoreUnlimited(t *testing.T) {
	st, err := NewMemStore(0)
	if err != nil {
		t.Fatal(err)
	}
	storeTest(t, st)
}

func BenchmarkMemStoreLRU(b *testing.B) {
	st, err := NewMemStore(0)
	if err != nil {
		b.Fatal(err)
	}
	storeBenchmark(b, st)
}

func BenchmarkMemStoreUnlimited(b *testing.B) {
	st, err := NewMemStore(10)
	if err != nil {
		b.Fatal(err)
	}
	storeBenchmark(b, st)
}
