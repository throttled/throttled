package store

import (
	"testing"
)

func TestMemStoreLRU(t *testing.T) {
	storeTest(t, NewMemStore(10))
}

func TestMemStoreUnlimited(t *testing.T) {
	storeTest(t, NewMemStore(0))
}

func BenchmarkMemStoreLRU(b *testing.B) {
	storeBenchmark(b, NewMemStore(0))
}

func BenchmarkMemStoreUnlimited(b *testing.B) {
	storeBenchmark(b, NewMemStore(10))
}
