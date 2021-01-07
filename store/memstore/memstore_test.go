package memstore_test

import (
	"testing"

	"github.com/throttled/throttled/v2/store/memstore"
	"github.com/throttled/throttled/v2/store/storetest"
)

func TestMemStoreLRU(t *testing.T) {
	st, err := memstore.NewCtx(10)
	if err != nil {
		t.Fatal(err)
	}
	storetest.TestGCRAStoreCtx(t, st)
}

func TestMemStoreUnlimited(t *testing.T) {
	st, err := memstore.NewCtx(10)
	if err != nil {
		t.Fatal(err)
	}
	storetest.TestGCRAStoreCtx(t, st)
}

func BenchmarkMemStoreLRU(b *testing.B) {
	st, err := memstore.NewCtx(10)
	if err != nil {
		b.Fatal(err)
	}
	storetest.BenchmarkGCRAStoreCtx(b, st)
}

func BenchmarkMemStoreUnlimited(b *testing.B) {
	st, err := memstore.NewCtx(0)
	if err != nil {
		b.Fatal(err)
	}
	storetest.BenchmarkGCRAStoreCtx(b, st)
}
