package go_cache

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
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

// BenchmarkGetHit tests the performance of the Get operation for cache hits.
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

// BenchmarkContainsExists tests the performance of the Contains operation for existing keys.
func BenchmarkContainsExists(b *testing.B) {
	ctx := context.Background()
	cache := NewMemoryCache[string, string](ctx, 5*time.Minute, 1*time.Minute, 1000)
	// Pre-populate cache
	for i := 0; i < 500; i++ {
		key := fmt.Sprintf("key-%d", i)
		cache.Set(key, "value", 0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%500)
		cache.Contains(key)
	}
}

// BenchmarkContainsNotExists tests the performance of the Contains operation for non-existent keys.
func BenchmarkContainsNotExists(b *testing.B) {
	ctx := context.Background()
	cache := NewMemoryCache[string, string](ctx, 5*time.Minute, 1*time.Minute, 1000)
	// Pre-populate cache
	for i := 0; i < 500; i++ {
		key := fmt.Sprintf("key-%d", i)
		cache.Set(key, "value", 0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i+1000)
		cache.Contains(key)
	}
}

// BenchmarkLen tests the performance of the Len operation.
func BenchmarkLen(b *testing.B) {
	ctx := context.Background()
	cache := NewMemoryCache[string, string](ctx, 5*time.Minute, 1*time.Minute, 1000)
	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		cache.Set(fmt.Sprintf("key-%d", i), "value", 0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Len()
	}
}

// BenchmarkMemoryCache_Fetch benchmarks the performance of the Fetch (Get) method in MemoryCache.
func BenchmarkMemoryCacheFetch(b *testing.B) {
	// Create a new MemoryCache instance with a TTL of 1 hour.
	cache := NewMemoryCache[string, int](context.Background(), 1*time.Hour, 1*time.Hour, 1000)

	// Populate the cache with benchmark data using goroutines.
	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		// Add an item to the cache in a goroutine to simulate concurrent access.
		go func(key string, value int, ttl time.Duration) {
			cache.Set(key, value, ttl)
			wg.Done()
		}(fmt.Sprintf("key%d", i), i, 10*time.Minute) // Each item has a TTL of 10 minutes.
	}

	// Wait for all goroutines to finish populating the cache.
	wg.Wait()

	// Reset the benchmark timer to exclude setup time.
	b.ResetTimer()

	// Benchmark the Fetch operation.
	for i := 0; i < b.N; i++ {
		// Retrieve an item from the cache with a unique key.
		cache.Get(fmt.Sprintf("key%d", i))
	}

	// Forcing the garbage collector (GC) to run to clear memory after benchmarking.
	runtime.GC()
}

// BenchmarkExpiration tests the performance of the expiration cleanup with a small dataset.
func BenchmarkExpiration(b *testing.B) {
	ctx := context.Background()
	cache := NewMemoryCache[string, string](ctx, 1*time.Millisecond, 1*time.Millisecond, 1000)
	// Pre-populate cache with expired items
	for i := 0; i < 1000; i++ {
		cache.Set(fmt.Sprintf("key-%d", i), "value", 1*time.Millisecond)
	}
	time.Sleep(2 * time.Millisecond) // Ensure items are expired

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.deleteExpiredData()
	}
}
