package throttled

import (
	"runtime"
	"testing"
)

func BenchmarkReadMemStats(b *testing.B) {
	var mem runtime.MemStats
	for i := 0; i < b.N; i++ {
		runtime.ReadMemStats(&mem)
	}
}
