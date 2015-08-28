package mem_test

import (
	"testing"

	"gopkg.in/throttled/throttled.v1/store/mem"
	"gopkg.in/throttled/throttled.v1/store/storetest"
)

func TestMemStoreLRU(t *testing.T) {
	st, err := mem.New(10)
	if err != nil {
		t.Fatal(err)
	}
	storetest.TestGCRAStore(t, st)
}

func TestMemStoreUnlimited(t *testing.T) {
	st, err := mem.New(10)
	if err != nil {
		t.Fatal(err)
	}
	storetest.TestGCRAStore(t, st)
}

func BenchmarkMemStoreLRU(b *testing.B) {
	st, err := mem.New(10)
	if err != nil {
		b.Fatal(err)
	}
	storetest.BenchmarkGCRAStore(b, st)
}

func BenchmarkMemStoreUnlimited(b *testing.B) {
	st, err := mem.New(0)
	if err != nil {
		b.Fatal(err)
	}
	storetest.BenchmarkGCRAStore(b, st)
}
