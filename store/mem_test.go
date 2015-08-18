package store_test

import (
	"testing"

	"gopkg.in/throttled/throttled.v0/store"
)

func TestMemStoreLRU(t *testing.T) {
	storeTest(t, store.NewMemStore(10))
}

func TestMemStoreUnlimited(t *testing.T) {
	storeTest(t, store.NewMemStore(0))
}

func BenchmarkMemStoreLRU(b *testing.B) {
	storeBenchmark(b, store.NewMemStore(0))
}

func BenchmarkMemStoreUnlimited(b *testing.B) {
	storeBenchmark(b, store.NewMemStore(10))
}
