package store

import (
	"math/rand"
	"strconv"
	"sync/atomic"
	"testing"
)

func storeTest(t *testing.T, st Store) {
	// SetNX on a new key
	want := int64(1)
	set, err := st.SetNX("foo", want)
	if err != nil {
		t.Fatal(err)
	}
	if !set {
		t.Errorf("expected SetNX on an empty key to succeed")
	}

	have, err := st.Get("foo")
	if err != nil {
		t.Fatal(err)
	}
	if have != want {
		t.Errorf("expected Get to return %d but got %d", want, have)
	}

	// SetNX on an existing key
	set, err = st.SetNX("foo", 123)
	if err != nil {
		t.Fatal(err)
	}
	if set {
		t.Errorf("expected SetNX on an existing key to fail")
	}

	have, err = st.Get("foo")
	if err != nil {
		t.Fatal(err)
	}
	if have != want {
		t.Errorf("expected Get to return %d but got %d", want, have)
	}

	// SetNX on a different key
	set, err = st.SetNX("bar", 456)
	if err != nil {
		t.Fatal(err)
	}
	if !set {
		t.Errorf("expected SetNX on an empty key to succeed")
	}

	// Returns the correct error on a missing key
	_, err = st.CompareAndSwap("baz", 1, 2)
	if want := ErrNoSuchKey; err != want {
		t.Errorf("expected CompareAndSwap to fail with %q but got %q", want, err)
	}

	// Test a successful CAS
	want = int64(2)
	swapped, err := st.CompareAndSwap("foo", 1, want)
	if err != nil {
		t.Fatal(err)
	}
	if !swapped {
		t.Errorf("expected CompareAndSwap to succeed")
	}

	have, err = st.Get("foo")
	if err != nil {
		t.Fatal(err)
	}
	if have != want {
		t.Errorf("expected Get to return %d but got %d", want, have)
	}

	// Test an unsuccessful CAS
	swapped, err = st.CompareAndSwap("foo", 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if swapped {
		t.Errorf("expected CompareAndSwap to fail")
	}

	have, err = st.Get("foo")
	if err != nil {
		t.Fatal(err)
	}
	if have != want {
		t.Errorf("expected Get to return %d but got %d", want, have)
	}
}

func storeBenchmark(b *testing.B, st Store) {
	seed := int64(42)
	var attempts, updates, evictions int64

	b.RunParallel(func(pb *testing.PB) {
		// We need atomic behavior around the RNG or go detects a race in the test
		delta := int64(1)
		seedValue := atomic.AddInt64(&seed, delta) - delta
		gen := rand.New(rand.NewSource(seedValue))

		for pb.Next() {
			key := strconv.FormatInt(gen.Int63n(50), 10)

			var v int64
			var updated bool

			v, err := st.Get(key)
			if err == ErrNoSuchKey {
				updated, err = st.SetNX(key, gen.Int63())
				if err != nil {
					b.Error(err)
				}
			} else if err != nil {
				b.Error(err)
			} else {
				updated, err = st.CompareAndSwap(key, v, gen.Int63())
				if err == ErrNoSuchKey {
					atomic.AddInt64(&evictions, 1)
				} else if err != nil {
					b.Error(err)
				}
			}

			atomic.AddInt64(&attempts, 1)
			if updated {
				atomic.AddInt64(&updates, 1)
			}
		}
	})

	b.Logf("%d/%d update operations succeeed (%d evicted)", updates, attempts, evictions)
}
