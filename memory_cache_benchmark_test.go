package go_cache

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"testing"
	"time"
)

func BenchmarkMemoryCacheSet(b *testing.B) {
	// Create a new MemoryCache instance with a TTL of 1 minute.
	cache := NewMemoryCache[string, int](context.Background(), 5*time.Minute, 1*time.Minute, 1000)

	// Reset the benchmark timer to exclude setup time.
	b.ReportAllocs()

	b.ResetTimer()
	// Run the benchmark to measure the performance of the Set method.
	for i := 0; i < b.N; i++ {
		// Set an item in the cache with a unique key and a TTL of 1 minute.
		cache.Set(fmt.Sprintf("key%d", i), i, 0)
	}

	// Forcing the garbage collector (GC) to run to clear memory after benchmarking.
	runtime.GC()
}

func BenchmarkMemoryCacheParallelSet(b *testing.B) {
	// Create a new MemoryCache instance with a TTL of 1 minute.
	cache := NewMemoryCache[string, int](context.Background(), 5*time.Minute, 1*time.Minute, 1000)

	// Reset the benchmark timer to exclude setup time.
	b.ReportAllocs()

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i := rand.Int()
			// Set an item in the cache with a unique key and a TTL of 1 minute.
			cache.Set(fmt.Sprintf("key%d", rand.Int()), i, 0)
		}
	})

	// Forcing the garbage collector (GC) to run to clear memory after benchmarking.
	runtime.GC()
}
