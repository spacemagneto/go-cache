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

func BenchmarkGetHit(b *testing.B) {
	ctx := context.Background()
	cache := NewMemoryCache[string, string](ctx, 5*time.Minute, 1*time.Minute, 1000)
	// Pre-populate cache with some items
	for i := 0; i < 500; i++ {
		key := fmt.Sprintf("key-%d", i)
		cache.Set(key, "value", 0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%500)
		cache.Get(key)
	}
}

// BenchmarkGetMiss tests the performance of the Get operation for cache misses.
func BenchmarkGetMiss(b *testing.B) {
	ctx := context.Background()
	cache := NewMemoryCache[string, string](ctx, 5*time.Minute, 1*time.Minute, 1000)
	// Pre-populate cache with some items
	for i := 0; i < 500; i++ {
		key := fmt.Sprintf("key-%d", i)
		cache.Set(key, "value", 0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i+1000) // Access non-existent keys
		cache.Get(key)
	}
}

// BenchmarkRemove tests the performance of the Remove operation.
func BenchmarkRemove(b *testing.B) {
	ctx := context.Background()
	cache := NewMemoryCache[string, string](ctx, 5*time.Minute, 1*time.Minute, 1000)
	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key-%d", i)
		cache.Set(key, "value", 0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%1000)
		cache.Remove(key)
	}
}
