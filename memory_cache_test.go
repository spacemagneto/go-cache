package go_cache

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMemoryCache(t *testing.T) {
	t.Parallel()

	// SetAndFetch tests the functionality of setting a key-value pair in the memory cache
	// and subsequently retrieving it. It validates that the cache correctly stores the key-value pair
	// and retrieves it before the time-to-live (TTL) expires. This test ensures that the caching
	// mechanism functions as expected, preserving data integrity and accessibility during its TTL.
	t.Run("SetAndFetch", func(t *testing.T) {
		// Create a new instance of MemoryCache with a background context, a global TTL of 5 seconds,
		// and a maximum capacity of 1000 entries. This initializes the memory cache for testing.
		cache := NewMemoryCache[string, int](context.Background(), 5*time.Second, 5*time.Second, 1000)

		// Insert a key-value pair into the cache. The key is "key1" and the value is 42.
		// The TTL for this entry is set to 1 second, meaning the key-value pair will expire
		// and be removed from the cache after 1 second.
		cache.Set("key1", 42, time.Second)

		// Retrieve the value associated with "key1" from the cache. If the key exists and
		// has not expired, the method returns the value and a boolean flag indicating success.
		element, ok := cache.Get("key1")

		// Verify that the retrieved element is not nil. This ensures the cache correctly stored
		// the key-value pair and returned a non-nil value for the existing key.
		assert.NotNil(t, element, "Expected element to be non-nil")

		// Confirm that the retrieved value is 42. This ensures the cache correctly preserved
		// the value associated with "key1" and returned the expected data.
		assert.Equal(t, 42, element, "Expected element to be 42")

		// Check that the boolean flag `ok` is true. This indicates that the key "key1" exists
		// in the cache and was successfully retrieved without any errors.
		assert.Equal(t, true, ok, "Expected key 'key1' to exist in cache")
	})

	// SetAndContains tests the functionality of setting key-value pairs in the memory cache
	// and checking their presence using the Contains method. It validates that the cache
	// correctly stores keys without and with TTL, ensures data retrieval is accurate, and
	// confirms the keys are properly tracked by the cache. This test ensures the cache's
	// key containment and retrieval mechanisms function as intended.
	t.Run("SetAndContains", func(t *testing.T) {
		// Create a context with cancellation capability for managing cache lifecycle.
		// This ensures the cache can be properly stopped after the test completes.
		ctx, cancel := context.WithCancel(context.Background())
		// Defer the cancellation of the context to ensure it is cleaned up after the test.
		// This prevents resource leaks in the case of errors or test completion.
		defer cancel()

		// Instantiate a new memory cache with a TTL of 5 seconds and a maximum capacity of 1000 items.
		// This cache will be used to test the Set and Contains methods.
		cache := NewMemoryCache[string, string](ctx, 5*time.Second, 5*time.Second, 1000)

		// Insert a key-value pair ("key1", "value1") into the cache without specifying a TTL.
		// This operation tests the cache's ability to store values without expiration.
		cache.Set("key1", "value1", 0)
		// Verify that the key "key1" exists in the cache using the Contains method.
		// This confirms that the cache correctly tracks inserted keys.
		assert.True(t, cache.Contains("key1"), "Expected cache to contain key 'key1'")

		// Retrieve the value associated with "key1" to confirm it was stored correctly.
		// The Get method should return the value and a boolean indicating the key exists.
		val, ok := cache.Get("key1")
		// Assert that the key "key1" exists in the cache.
		// The boolean result from Get should be true for existing keys.
		assert.True(t, ok, "Expected key 'key1' to exist in cache")
		// Assert that the value retrieved for "key1" matches the expected value "value1".
		// This ensures the cache stored the correct value.
		assert.Equal(t, "value1", val, "Expected value for 'key1' to be 'value1'")

		// Insert another key-value pair ("key2", "value2") into the cache with a TTL of 1 second.
		// This tests the cache's ability to handle values with expiration times.
		cache.Set("key2", "value2", time.Second)
		// Verify that the key "key2" exists in the cache immediately after insertion.
		// The Contains method should report true for recently added keys.
		assert.True(t, cache.Contains("key2"), "Expected cache to contain key 'key2'")

		// Retrieve the value associated with "key2" to confirm it was stored correctly.
		// The Get method should return the value and a boolean indicating the key exists.
		_, ok = cache.Get("key2")
		// Assert that the key "key2" exists in the cache.
		// The boolean result from Get should be true for existing keys.
		assert.True(t, ok, "Expected key 'key2' to exist in cache")
	})

	// Double tests the behavior of the MemoryCache when the same key is set multiple times
	// with different values. It ensures that the cache overwrites the existing value for a
	// key when a new value is set and correctly retrieves the latest value. This test validates
	// the cache's functionality in handling updates to existing keys.
	t.Run("Double", func(t *testing.T) {
		// Create a new instance of MemoryCache with a TTL of 5 seconds and a maximum size of 1000.
		// This cache will store key-value pairs of string keys and integer values.
		cache := NewMemoryCache[string, int](context.Background(), 5*time.Second, 5*time.Second, 1000)

		// Set the key "key1" with the value 42 and a TTL of 3 seconds.
		// This operation initializes the cache entry for "key1".
		cache.Set("key1", 42, 3*time.Second)
		// Overwrite the key "key1" with a new value of 57 and the same TTL of 3 seconds.
		// This tests the cache's ability to update the value for an existing key.
		cache.Set("key1", 57, 3*time.Second)

		// Retrieve the value associated with "key1" from the cache.
		// This ensures the cache returns the most recently set value for the key.
		element, ok := cache.Get("key1")

		// Assert that the retrieved element is not nil, indicating that the key exists in the cache.
		// This verifies that the cache correctly tracks the updated key.
		assert.NotNil(t, element, "Expected element to be non-nil")
		// Assert that the retrieved value matches the most recent value (57) set for "key1".
		// This confirms the cache correctly overwrote the old value with the new one.
		assert.Equal(t, 57, element, "Expected element to be 57")
		// Assert that the key "key1" exists in the cache.
		// The boolean return value of the Get method should indicate the key's presence.
		assert.True(t, ok, "Expected key 'key1' to exist in cache")
	})

	// FetchNotFound tests the behavior of the MemoryCache when attempting to retrieve a key
	// that does not exist. It ensures that the cache correctly returns a failure indicator
	// when the requested key is not found, validating the cache's handling of nonexistent keys.
	t.Run("FetchNotFound", func(t *testing.T) {
		// Create a new instance of MemoryCache with a TTL of 5 seconds and a maximum size of 1000.
		// This cache will store key-value pairs with string keys and integer values.
		cache := NewMemoryCache[string, int](context.Background(), 5*time.Second, 5*time.Second, 1000)

		// Attempt to retrieve the value associated with the key "nonexistent" from the cache.
		// Since this key has not been set, the cache is expected to return a failure indicator.
		_, ok := cache.Get("nonexistent")
		// Assert that the cache indicates the key "nonexistent" does not exist.
		// This verifies that the cache properly handles requests for missing keys.
		assert.Equal(t, false, ok, "Expected key 'nonexistent' not to exist in cache")
	})

	// Remove tests the functionality of deleting a key-value pair from the memory cache.
	// It verifies that after removing a key, it cannot be fetched from the cache,
	// ensuring the cache's state is accurately updated after the removal.
	t.Run("Remove", func(t *testing.T) {
		// Create a new instance of MemoryCache with a background context, a global TTL of 5 seconds,
		// and a maximum capacity of 1000 entries. This initializes the cache for this test.
		cache := NewMemoryCache[string, int](context.Background(), 5*time.Second, 5*time.Second, 1000)

		// Insert a key-value pair into the cache. The key is "key1" and the value is 42.
		// The TTL for this entry is set to 1 second, meaning the key-value pair will expire quickly.
		cache.Set("key1", 42, time.Second)

		// Remove the key "key1" from the cache. The Remove method is expected to return true,
		// indicating that the key existed and was successfully removed from the cache.
		ok := cache.Remove("key1")
		// Verify that the Remove method returned true, confirming the key was removed successfully.
		// This ensures that the cache correctly updated its state by removing the key.
		assert.Equal(t, true, ok, "Expected Remove to return true")

		// Attempt to fetch the value associated with "key1" after it has been removed.
		// The Get method should indicate that the key no longer exists in the cache.
		_, ok = cache.Get("key1")
		// Verify that the key "key1" no longer exists in the cache.
		// This confirms the Remove method's correctness in deleting the key-value pair.
		assert.False(t, ok, "Expected key 'key1' to be removed from cache")
	})

	// RemoveAndContains tests the combined functionality of removing a key-value pair
	// from the memory cache and checking its existence afterward using the Contains method.
	// This test ensures the Remove method correctly updates the cache state and that
	// the Contains method accurately reflects the absence of the removed key.
	t.Run("RemoveAndContains", func(t *testing.T) {
		// Create a context with cancellation to manage the cache's lifecycle during the test.
		// The cancel function is deferred to ensure the context is properly closed after the test.
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Initialize a new instance of MemoryCache with the context, a global TTL of 5 seconds,
		// and a maximum capacity of 1000 entries. This prepares the cache for testing.
		cache := NewMemoryCache[string, string](ctx, 5*time.Second, 5*time.Second, 1000)

		// Attempt to remove a non-existent key "nonexistent" from the cache.
		// This checks if the Remove method correctly identifies keys that do not exist.
		assert.False(t, cache.Remove("nonexistent"), "Expected Remove to return false for non-existent key")

		// Add a key-value pair to the cache with "key1" as the key and "value1" as the value.
		// The TTL is set to 0, which indicates the entry should not expire automatically.
		cache.Set("key1", "value1", 0)

		// Remove the key "key1" from the cache. The Remove method is expected to return true,
		// indicating the key was successfully removed from the cache's internal state.
		assert.True(t, cache.Remove("key1"), "Expected Remove to return true for existing key")

		// Check if the key "key1" exists in the cache after being removed.
		// The Contains method should return false, confirming the key was deleted.
		assert.False(t, cache.Contains("key1"), "Expected key 'key1' to be removed from cache")
	})

	// RemoveNotFound tests the behavior of the Remove method when attempting to remove a key
	// that does not exist in the memory cache. This test ensures that the method handles
	// non-existent keys gracefully and returns false, indicating no removal was performed.
	t.Run("RemoveNotFound", func(t *testing.T) {
		// Initialize a new instance of MemoryCache with a background context.
		// The cache has a global TTL of 5 seconds and a maximum capacity of 1000 entries,
		// providing a consistent environment for testing.
		cache := NewMemoryCache[string, int](context.Background(), 5*time.Second, 5*time.Second, 1000)

		// Attempt to remove the key "nonexistent" from the cache.
		// This key does not exist in the cache, so the Remove method should return false.
		ok := cache.Remove("nonexistent")

		// Verify that the Remove method returns false.
		// This confirms that the method correctly identified the key as non-existent
		// and did not perform any removal operations.
		assert.Equal(t, false, ok, "Expected Remove to return false for non-existent key")
	})

	// Expiration tests the cache's ability to correctly handle item expiration based on TTL.
	// It ensures that an item is automatically removed from the cache after its TTL expires,
	// validating the eviction mechanism and maintaining cache integrity.
	t.Run("Expiration", func(t *testing.T) {
		// Create a new MemoryCache instance with a background context.
		// The global TTL for the cache is set to 150 milliseconds, and the maximum capacity is 1000 entries.
		cache := NewMemoryCache[string, int](context.Background(), 150*time.Millisecond, 150*time.Millisecond, 1000)

		// Add an item to the cache with the key "key1" and value 42.
		// The TTL for this specific entry is set to 100 milliseconds, after which it will expire.
		cache.Set("key1", 42, 100*time.Millisecond)

		// Pause execution for 300 milliseconds to ensure that the item's TTL has expired.
		// This simulates a delay to test whether the cache correctly evicts expired items.
		<-time.After(160 * time.Millisecond)

		// Attempt to retrieve the item associated with the key "key1" from the cache.
		// Since the TTL has expired, the item should no longer be present in the cache.
		_, ok := cache.Get("key1")

		// Verify that the item has been successfully evicted from the cache.
		// The `Get` method should return false, indicating the item is no longer available.
		assert.False(t, ok, "Expected the item to be expired and removed from the cache")
	})

	// Len tests the functionality of the Len method in the MemoryCache.
	// This test ensures that the method correctly reports the number of entries in the cache
	// after performing operations such as adding and removing items.
	t.Run("Len", func(t *testing.T) {
		// Create a background context for the cache operations.
		// This context provides a way to manage the lifecycle of the MemoryCache instance.
		ctx := context.Background()

		// Initialize a new MemoryCache instance with a TTL of 5 minutes and a capacity of 1000 entries.
		// This setup allows us to test the cache's behavior under different operations.
		cache := NewMemoryCache[string, string](ctx, 5*time.Minute, 5*time.Minute, 1000)

		// Check the initial length of the cache, which should be zero.
		// This ensures that a newly created cache has no stored entries.
		assert.Equal(t, 0, cache.Len(), "Len() on empty cache should be 0")

		// Add the first item to the cache with key "key1" and value "value1".
		// The TTL is set to 0, meaning the item will not expire unless explicitly removed.
		cache.Set("key1", "value1", 0)

		// Add a second item to the cache with key "key2" and value "value2".
		// This increases the total number of items in the cache to two.
		cache.Set("key2", "value2", 0)

		// Verify that the cache reports a length of two after adding the items.
		// This checks that the Len method reflects the current state of the cache accurately.
		assert.Equal(t, 2, cache.Len(), "Len() after adding 2 items should be 2")

		// Remove the item with key "key1" from the cache.
		// This operation reduces the number of items in the cache by one.
		cache.Remove("key1")

		// Verify that the cache reports a length of one after removing an item.
		// This confirms that the Len method updates correctly when items are removed.
		assert.Equal(t, 1, cache.Len(), "Len() after removing 1 item should be 1")
	})

	// MixedExpiredAndNonExpired tests the cache’s ability to correctly remove expired items
	// while preserving non-expired items. It verifies that after setting one key with a very short TTL
	// and another with a long TTL, only the expired key is removed when cleanup is triggered.
	// The test confirms that internal cache structures such as size counter, LRU list, and expiration heap
	// remain consistent and accurate after expired entries are deleted.
	t.Run("MixedExpiredAndNonExpired", func(t *testing.T) {
		// Create a background context for the cache.
		// This provides a context without cancellation for testing.
		ctx := context.Background()
		// Initialize a new MemoryCache with string keys and string values.
		// The cache is configured with a 1ms cleanup interval and 1000 capacity.
		cache := NewMemoryCache[string, string](ctx, 1*time.Millisecond, 1*time.Minute, 1000)

		// Set an expired item with a 1ms TTL.
		// This adds "key1" to the cache, expiring immediately.
		cache.Set("key1", "value1", 1*time.Millisecond)
		// Set a non-expired item with a 1-hour TTL.
		// This adds "key2" to the cache, which should persist after cleanup.
		cache.Set("key2", "value2", 1*time.Hour)
		// Wait briefly to ensure key1 expires.
		// This allows the 1ms TTL to pass for key1.
		time.Sleep(2 * time.Millisecond)
		// Call deleteExpiredData to remove expired items.
		// This processes the expirationHeap and updates the cache.
		cache.deleteExpiredData()
		// Check if key1 was removed from the cache.
		// This tests if the expired item was deleted from the items map.
		_, ok := cache.Get("key1")
		// Assert that key1 is no longer present.
		// This confirms that deleteExpiredData removed the expired item.
		assert.False(t, ok, "Expected Get(key1) to return false after expiration")
		// Check if key2 is still present in the cache.
		// This tests if the non-expired item was preserved.
		value, ok := cache.Get("key2")
		// Assert that key2 is present.
		// This verifies that non-expired items are not removed.
		assert.True(t, ok, "Expected Get(key2) to return true for non-expired item")
		// Assert that key2’s value is correct.
		// This confirms that the item’s data was preserved.
		assert.Equal(t, "value2", value, "Expected Get(key2) to return value2")
		// Verify the cache’s size counter.
		// This should be 1, as only key2 remains.
		assert.Equal(t, int32(1), cache.size.Load(), "Expected size to be 1 after removing expired item")
		// Verify the LRU list length.
		// This should be 1, as only key2’s element remains.
		assert.Equal(t, 1, cache.list.Len(), "Expected LRU list length to be 1 after removing expired item")
		// Verify the expirationHeap length.
		// This should be 1, as only key2’s expiration timestamp remains.
		assert.Equal(t, 1, cache.expirationHeap.Len(), "Expected expirationHeap length to be 1 after removing expired item")
	})

	// ItemInHeapNotInItems tests the cache’s behavior when an item exists in the expiration heap
	// but is missing from the items map and LRU list. This simulates a state where the
	// item has been partially removed but still lingers in the expiration heap.
	// The test verifies that the deleteExpiredData method correctly cleans up the heap entry
	// without causing errors or inconsistencies in the cache's internal state.
	t.Run("ItemInHeapNotInItems", func(t *testing.T) {
		// Create a background context for the cache.
		// This provides a context without cancellation for testing.
		ctx := context.Background()
		// Initialize a new MemoryCache with string keys and string values.
		// The cache is configured with a 1ms cleanup interval and 1000 capacity.
		cache := NewMemoryCache[string, string](ctx, 1*time.Millisecond, 1*time.Minute, 1000)

		// Set an expired item with a 1ms TTL.
		// This adds "key1" to the cache, expiring immediately.
		cache.Set("key1", "value1", 1*time.Millisecond)

		// Manually remove key1 from the items map and LRU list.
		// This simulates a scenario where the item is only in the expirationHeap.
		cache.mutex.Lock()
		// Remove key1 from the items map.
		// This deletes the map entry but leaves the item in the heap.
		if element, ok := cache.items["key1"]; ok {
			// Remove the item from the LRU list.
			// This ensures the item is fully removed from the cache’s data structures.
			cache.list.Remove(element)
			// Delete the item from the items map.
			// This completes the manual removal.
			delete(cache.items, "key1")
			// Decrement the size counter.
			// This reflects the removal of the item.
			cache.size.Add(-1)
		}

		// Release the lock after manual removal.
		// This allows deleteExpiredData to proceed.
		cache.mutex.Unlock()
		// Wait briefly to ensure key1’s TTL expires.
		// This ensures the item in the heap is considered expired.
		time.Sleep(2 * time.Millisecond)

		// Call deleteExpiredData to process the expired item.
		// This should remove key1 from the expirationHeap.
		cache.deleteExpiredData()

		// Verify the cache’s size counter.
		// This should be 0, as no items remain.
		assert.Equal(t, int32(0), cache.size.Load(), "Expected size to be 0 after processing missing item")
		// Verify the LRU list length.
		// This should be 0, as no items remain in the list.
		assert.Equal(t, 0, cache.list.Len(), "Expected LRU list length to be 0 after processing missing item")
		// Verify the expirationHeap length.
		// This should be 0, as the expired item was removed from the heap.
		assert.Equal(t, 0, cache.expirationHeap.Len(), "Expected expirationHeap length to be 0 after processing missing item")
	})

	// ContextCancellation tests the MemoryCache behavior when its context is canceled.
	// It verifies that cache items persist after the cleanup goroutine stops due to cancellation.
	// The test ensures no unintended cleanup occurs, checking cache state with assertions.
	// It confirms the cache’s size, LRU list, and expiration heap remain intact post-cancellation.
	t.Run("ContextCancellation", func(t *testing.T) {
		// Create a cancelable context for the cache.
		// This allows testing context cancellation behavior.
		ctx, cancel := context.WithCancel(context.Background())

		// Initialize a new MemoryCache with string keys and string values.
		// The cache uses a 1ms cleanup interval to trigger frequent collector ticks.
		cache := NewMemoryCache[string, string](ctx, 1*time.Millisecond, 1*time.Minute, 1000)
		// Set a non-expired item with a 1-hour TTL.
		// This adds "key1" to the cache to test post-cancellation state.
		cache.Set("key1", "value1", 1*time.Hour)

		// Cancel the context immediately to stop the collector.
		// This tests rapid shutdown before significant cleanup.
		cancel()

		// Wait briefly to allow the collector to exit.
		// This ensures the goroutine processes the cancellation.
		time.Sleep(5 * time.Millisecond)
		// Acquire a read lock to safely check the cache state.
		// This ensures thread-safe access to the cache’s data structures.
		cache.mutex.RLock()
		// Verify that key1 is still present in the cache.
		// This tests if the item persists after collector shutdown.
		_, ok := cache.items["key1"]
		// Assert that key1 is present in the items map.
		// This confirms that cancellation did not trigger unintended cleanup.
		assert.True(t, ok, "Expected key1 to remain in items map after context cancellation")
		// Verify the cache’s size counter.
		// This should be 1, as no cleanup occurred after cancellation.
		assert.Equal(t, int32(1), cache.size.Load(), "Expected size to be 1 after context cancellation")
		// Verify the LRU list length.
		// This should be 1, as key1’s element remains in the list.
		assert.Equal(t, 1, cache.list.Len(), "Expected LRU list length to be 1 after context cancellation")
		// Verify the expirationHeap length.
		// This should be 1, as key1’s expiration timestamp remains.
		assert.Equal(t, 1, cache.expirationHeap.Len(), "Expected expirationHeap length to be 1 after context cancellation")
		// Release the read lock after checking the cache state.
		// This allows further operations on the cache.
		cache.mutex.RUnlock()
	})
}

// TestConcurrentAccess tests thread-safety under concurrent Set, Get, and Remove.
// TestConcurrentAccess tests the MemoryCache's ability to handle concurrent access.
// It validates the thread-safety of the cache during simultaneous Set, Get, and Remove operations
// by multiple goroutines. The test ensures that no race conditions or panics occur under high concurrency.
func TestConcurrentAccess(t *testing.T) {
	// Create a background context to manage the lifecycle of the MemoryCache instance.
	// This context is passed to the MemoryCache to ensure operations can be canceled if needed.
	ctx := context.Background()

	// Initialize a new MemoryCache with a TTL of 5 minutes and a capacity of 1000 entries.
	// This instance will be used to test concurrent access.
	cache := NewMemoryCache[string, string](ctx, 5*time.Minute, 5*time.Minute, 1000)

	// Define a WaitGroup to synchronize the completion of all goroutines.
	// This ensures that the test waits for all concurrent operations to finish.
	var wg sync.WaitGroup

	// Specify the number of worker goroutines that will perform operations on the cache.
	const numWorkers = 10

	// Specify the number of iterations each worker will perform.
	// This determines the workload for each goroutine.
	const iterations = 1000

	// Perform concurrent Set operations using multiple worker goroutines.
	for i := 0; i < numWorkers; i++ {
		// Increment the WaitGroup counter for each worker.
		wg.Add(1)

		// Launch a goroutine to perform Set operations.
		go func(id int) {
			// Decrement the WaitGroup counter when the goroutine completes.
			defer wg.Done()

			// Perform a series of Set operations using unique keys for each iteration.
			for j := 0; j < iterations; j++ {
				// Generate a unique key based on the worker ID and iteration index.
				key := fmt.Sprintf("key-%d-%d", id, j)

				// Store a value in the cache with no TTL, ensuring it persists unless explicitly removed.
				cache.Set(key, fmt.Sprintf("value-%d", j), 0)
			}
		}(i)
	}

	// Perform concurrent Get operations using multiple worker goroutines.
	for i := 0; i < numWorkers; i++ {
		// Increment the WaitGroup counter for each worker.
		wg.Add(1)

		// Launch a goroutine to perform Get operations.
		go func(id int) {
			// Decrement the WaitGroup counter when the goroutine completes.
			defer wg.Done()

			// Perform a series of Get operations using the same keys used in Set operations.
			for j := 0; j < iterations; j++ {
				// Generate a unique key based on the worker ID and iteration index.
				key := fmt.Sprintf("key-%d-%d", id, j)

				// Attempt to retrieve the value associated with the key from the cache.
				cache.Get(key)
			}
		}(i)
	}

	// Perform concurrent Remove operations using multiple worker goroutines.
	for i := 0; i < numWorkers; i++ {
		// Increment the WaitGroup counter for each worker.
		wg.Add(1)

		// Launch a goroutine to perform Remove operations.
		go func(id int) {
			// Decrement the WaitGroup counter when the goroutine completes.
			defer wg.Done()

			// Perform a series of Remove operations using the same keys used in Set operations.
			for j := 0; j < iterations; j++ {
				// Generate a unique key based on the worker ID and iteration index.
				key := fmt.Sprintf("key-%d-%d", id, j)

				// Remove the entry associated with the key from the cache.
				cache.Remove(key)
			}
		}(i)
	}

	// Wait for all goroutines to complete their operations.
	// This ensures that the test proceeds only after all Set, Get, and Remove operations are finished.
	wg.Wait()

	// Verify the final state of the cache. It should have fewer items than the total number of operations.
	// This ensures that the cache handles concurrent access safely without exceeding expected capacity.
	assert.LessOrEqual(t, cache.Len(), numWorkers*iterations, "Len() after concurrent access should not be excessively large")
}

func TestMemoryCacheConcurrency(t *testing.T) {
	// Mark the test to be run in parallel with other tests.
	t.Parallel()

	// Create a cancellable context to control the cache's lifecycle.
	ctx, cancel := context.WithCancel(context.Background())
	// Ensure the context is cancelled at the end of the test.
	defer cancel()

	// Initialize a new memory cache with the context and a default TTL.
	cache := NewMemoryCache[string, string](ctx, DefaultTTL, 1*time.Minute, 1000)
	// Number of concurrent goroutines to spawn.
	const numGoroutines = 100

	// Create a wait group to synchronize the completion of all goroutines.
	wg := &sync.WaitGroup{}
	// Ensure all goroutines have completed before the test ends.
	defer wg.Wait()

	// Subtest for concurrent set and get operations.
	// TestConcurrentSetAndGet tests the concurrent setting and getting of cache values.
	// It ensures that the cache can handle multiple goroutines setting and getting values concurrently.
	t.Run("TestConcurrentSetAndGet", func(t *testing.T) {
		// Loop to start multiple goroutines.
		for i := 0; i < numGoroutines; i++ {
			// Increment the wait group counter for each goroutine.
			wg.Add(1)

			// Launch a goroutine to perform set and get operations.
			go func(i int) {
				// Decrement the counter when the goroutine completes.
				defer wg.Done()

				// Set a key-value pair in the cache.
				cache.Set(fmt.Sprintf("key%d", i), fmt.Sprintf("value%d", i), 0)

				// Retrieve the value from the cache.
				val, ok := cache.Get(fmt.Sprintf("key%d", i))

				// Verify that the value was retrieved successfully.
				assert.True(t, ok, fmt.Sprintf("Expected key key %d to exist in cache", i))
				assert.Equal(t, fmt.Sprintf("value%d", i), val, fmt.Sprintf("Expected value for key key %d to be value %d", i, i))
			}(i)
		}

		// Allow time for all goroutines to complete their operations.
		time.Sleep(2 * time.Second)
	})

	// Subtest for concurrent remove operations.
	// TestConcurrentRemove tests the concurrent removal of cache values.
	// It ensures that the cache can handle multiple goroutines removing values concurrently.
	t.Run("TestConcurrentRemove", func(t *testing.T) {
		// Loop to start multiple goroutines.
		for i := 0; i < numGoroutines; i++ {
			// Increment the wait group counter for each goroutine.
			wg.Add(1)

			// Launch a goroutine to perform set and remove operations.
			go func(i int) {
				// Decrement the counter when the goroutine completes.
				defer wg.Done()

				// Set a key-value pair in the cache.
				cache.Set(fmt.Sprintf("key%d", i), fmt.Sprintf("value%d", i), 0)

				// Remove the key from the cache.
				cache.Remove(fmt.Sprintf("key%d", i))

				// Attempt to retrieve the value from the cache.
				_, ok := cache.Get(fmt.Sprintf("key%d", i))

				// Verify that the value was not found in the cache.
				assert.False(t, ok, fmt.Sprintf("Expected key key%d to be removed from cache", i))
			}(i)
		}

		// Allow time for all goroutines to complete their operations.
		time.Sleep(2 * time.Second)
	})
}
