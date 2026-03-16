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

const (
	longTTL             = 7 * 24 * time.Hour
	longCleanupInterval = 7 * 24 * time.Hour
)

// Helper to create large values without escaping to heap in some cases
func makeValue(size int64) []byte {
	return make([]byte, size) // will escape, but that's fine for realistic benchmark
}

// BenchmarkMemoryCacheSetInteger-12    	 5 726 874	       197.6 ns/op	     183 B/op	       3 allocs/op
func BenchmarkMemoryCacheSetInteger(b *testing.B) {
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

// BenchmarkMemoryCache_Set_1MB
// Tests Set performance with 1 MB values
// BenchmarkMemoryCache_Set_1MB-12    	  234 867	      6205 ns/op	 1048729 B/op	       3 allocs/op
// BenchmarkMemoryCache_Set_1MB-12    	  281 506	      5307 ns/op	 1048730 B/op	       3 allocs/op
// BenchmarkMemoryCache_Set_1MB-12    	  119 358	     18056 ns/op	 1048805 B/op	       5 allocs/op
// BenchmarkMemoryCache_Set_1MB-12    	  249 176	      6550 ns/op	 1048735 B/op	       3 allocs/op
func BenchmarkMemoryCache_Set_1MB(b *testing.B) {
	const valueSize = 1 << 20 // 1 MB

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Limit cache to ~5–10 GB max to prevent OOM on typical machines
	cache := NewMemoryCache[int, []byte](ctx, longTTL, longCleanupInterval, 5000) // ~5 GB max

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cache.Set(i, makeValue(valueSize), 0)
	}
}

// BenchmarkMemoryCache_Set_10MB
// Tests Set performance with 10 MB values
// BenchmarkMemoryCache_Set_10MB-12    	   87991	    134574 ns/op	10485912 B/op	       3 allocs/op
func BenchmarkMemoryCache_Set_10MB(b *testing.B) {
	const valueSize = 10 << 20 // 10 MB

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Only ~500 items × 10 MB = 5 GB max
	cache := NewMemoryCache[int, []byte](ctx, longTTL, longCleanupInterval, 500)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cache.Set(i, makeValue(valueSize), 0)
	}
}

// BenchmarkMemoryCache_Set_50MB
// Tests Set performance with 50 MB values
func BenchmarkMemoryCache_Set_50MB(b *testing.B) {
	const valueSize = 50 << 20 // 50 MB

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cache := NewMemoryCache[int, []byte](ctx, longTTL, longCleanupInterval, 5000)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cache.Set(i, makeValue(valueSize), 0)
	}
}

const (
	benchCacheSize = 200_000
	benchWarmup    = 100_000
)

// Benchmark comparing your current xsync.RBMutex version vs original sync.RWMutex
// BenchmarkMemoryCache_GetHeavy/xsync.RBMutex-12         	 4 454 023	       279.6 ns/op	      11 B/op	       0 allocs/op
func BenchmarkMemoryCache_GetHeavy(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// === Your current version with xsync.RBMutex ===
	cacheX := NewMemoryCache[int, int](ctx, 10*time.Minute, 1*time.Minute, benchCacheSize)

	// Warm-up
	for i := 0; i < benchWarmup; i++ {
		cacheX.Set(i, i, 10*time.Minute)
	}

	// Reset the benchmark timer to exclude setup time.

	b.ResetTimer()

	b.Run("xsync.RBMutex", func(b *testing.B) {
		b.ReportAllocs()

		b.SetParallelism(8) // adjust to your core count

		b.RunParallel(func(pb *testing.PB) {
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			for pb.Next() {
				key := r.Intn(benchWarmup * 2) // ~50% hit rate

				// 92% Get, 8% Set — realistic cache workload
				if r.Intn(100) < 92 {
					cacheX.Get(key)
				} else {
					cacheX.Set(key+10000000, 999, 10*time.Minute)
				}
			}
		})
	})
}

// BenchmarkMemoryCacheSet-12    	 3 069 034	       384.0 ns/op	     309 B/op	       7 allocs/op
func BenchmarkMemoryCacheSet(b *testing.B) {
	// Create a new MemoryCache instance with a TTL of 1 minute.
	cache := NewMemoryCache[string, []byte](context.Background(), 5*time.Minute, 1*time.Minute, 10000000)

	// Reset the benchmark timer to exclude setup time.
	b.ReportAllocs()

	b.ResetTimer()
	// Run the benchmark to measure the performance of the Set method.
	for i := 0; i < b.N; i++ {
		value := []byte(fmt.Sprintf("value%d", i))
		// Set an item in the cache with a unique key and a TTL of 1 minute.
		cache.Set(fmt.Sprintf("key%d", i), value, 0)
	}

	// Forcing the garbage collector (GC) to run to clear memory after benchmarking.
	runtime.GC()
}

// BenchmarkMemoryCacheParallelSet-12    	 2 178 937	       547.4 ns/op	     168 B/op	       4 allocs/op
// BenchmarkMemoryCacheParallelSet-12    	 2 382 235	       541.7 ns/op	     284 B/op	       4 allocs/op
func BenchmarkMemoryCacheParallelSet(b *testing.B) {
	// Create a new MemoryCache instance with a TTL of 1 minute.
	cache := NewMemoryCache[string, int](context.Background(), 5*time.Minute, 1*time.Minute, 10000000)

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

// BenchmarkContainsExists tests the performance of the Contains operation for existing keys.
// BenchmarkContainsExists-12    	23883042	        50.39 ns/op
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
