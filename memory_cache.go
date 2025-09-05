package go_cache

import (
	"container/heap"
	"container/list"
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
)

// DefaultTTL is the default time-to-live duration for cache items.
const DefaultTTL = 1 * time.Hour

// MemoryCache represents an in-memory cache with TTL support and LRU eviction.
type MemoryCache[K comparable, V any] struct {
	parentCtx           context.Context      // Parent context to manage goroutines
	list                *list.List           // Doubly-linked list for LRU ordering
	items               map[K]*list.Element  // Map for quick lookups
	expirationHeap      ExpirationHeap[K, V] // Min-heap for expiration tracking
	wg                  sync.WaitGroup       // WaitGroup for managing goroutines
	ttl                 time.Duration        // Default time-to-live duration
	maxItems            int                  // Maximum number of items in the cache (0 = unlimited)
	size                atomic.Int32         // Atomic counter for cache size
	mutex               sync.RWMutex         // Mutex for thread safety
	group               singleflight.Group   // Singleflight for cache stampede prevention
	expireCheckInterval time.Duration        // Interval for cleaning up expired items
}

// NewMemoryCache creates a new MemoryCache instance with the specified TTL and max items.
// It starts a background goroutine to periodically delete expired items.
func NewMemoryCache[K comparable, V any](ctx context.Context, ttl, expireCheckInterval time.Duration, maxItems int) *MemoryCache[K, V] {
	if ttl == 0 {
		ttl = DefaultTTL
	}

	if expireCheckInterval == 0 {
		expireCheckInterval = DefaultTTL
	}

	cache := &MemoryCache[K, V]{parentCtx: ctx, list: list.New(), items: make(map[K]*list.Element), expirationHeap: make(ExpirationHeap[K, V], 0), ttl: ttl, maxItems: maxItems, expireCheckInterval: expireCheckInterval}

	// Initialize the expiration heap for the cache.
	// This ensures that the heap is properly set up for managing expiration times.
	heap.Init(&cache.expirationHeap)

	// Increment the wait group counter to indicate that a new goroutine is starting.
	// This prevents the program from exiting until this goroutine completes.
	cache.wg.Add(1)

	// Start a new goroutine to run the collector function.
	// The collector is responsible for managing expired cache entries.
	go func() {
		// Ensure the wait group counter is decremented when the goroutine exits.
		// This signals that the goroutine has finished its work.
		defer cache.wg.Done()

		// Call the collector method to handle cache entry expiration.
		// The collector method runs continuously to remove expired entries from the cache.
		cache.collector()
	}()

	return cache
}

// Set adds or updates an item in the cache with the specified key, value, and TTL.
// If the cache exceeds its maximum allowed items, the least recently used item is evicted.
// This method ensures that items are stored with proper expiration times and maintains
// the order of usage to support LRU eviction.
func (m *MemoryCache[K, V]) Set(key K, value V, ttl time.Duration) {
	// If TTL is zero, set it to the default TTL value.
	if ttl == 0 {
		ttl = m.ttl
	}

	// Create a new cache item with the specified key, value, and expiration time.
	// The expiration time is calculated based on the current time plus the TTL.
	item := &Item[K, V]{Key: key, Value: value, ExpiresAt: time.Now().Add(ttl)}

	// Acquire a write lock to ensure thread safety while updating the cache.
	// This prevents other goroutines from modifying the cache while this operation is in progress.
	m.mutex.Lock()
	// Ensure that the lock is released when the function completes, even if an error occurs or an early return happens.
	defer m.mutex.Unlock()

	// Check if the key already exists in the cache by looking it up in the items map.
	// If it exists, remove the corresponding element from the linked list.
	if element, ok := m.items[key]; ok {
		// Remove the existing element from the doubly linked list (m.list).
		m.list.Remove(element)
		// Decrement the size counter of the cache, as the old item is removed.
		m.size.Add(-1)

		// Note: The existing item is not removed from the expiration heap at this point.
		// It will eventually be replaced during heap operations.
	}

	// Check if the cache has reached its maximum capacity based on the maxItems setting.
	// If the cache is full, evict the least recently used (LRU) item to make space.
	if m.maxItems > 0 && int(m.size.Load()) >= m.maxItems {
		// Retrieve the least recently used item from the back of the doubly linked list.
		// This item has the lowest priority for retention in the cache.
		if element := m.list.Back(); element != nil {
			// Cast the retrieved element's value to the cache item type.
			// This step ensures the value can be manipulated correctly.
			oldItem := element.Value.(*Item[K, V])
			// Remove the least recently used item from the doubly linked list.
			// This operation updates the order of items in the cache.
			m.list.Remove(element)
			// Delete the removed item's entry from the items map using its key.
			// This ensures that the cache remains consistent with its internal structures.
			delete(m.items, oldItem.Key)
			// Decrement the size counter of the cache after removing the item.
			// The size counter must remain accurate to reflect the current state of the cache.
			m.size.Add(-1)

			// Note: The old item remains in the expiration heap and will be cleaned up later by a collector.
		}
	}

	// Insert the new item at the front of the doubly linked list.
	// Placing it at the front marks it as the most recently used item.
	element := m.list.PushFront(item)
	// Map the key to the newly added element in the items map.
	// This allows for quick lookup of the item based on its key.
	m.items[key] = element
	// Push the new cache item onto the expiration heap for expiration management.
	// The expiration heap ensures items expire in the correct order as their TTL elapses.
	heap.Push(&m.expirationHeap, item)

	// Increment the size counter to reflect the addition of the new cache item.
	// This maintains an accurate count of the items currently stored in the cache.
	m.size.Add(1)
}

// Get retrieves an item from the cache by its key.
// If the item is found and has not expired, it is returned along with a boolean true.
// If the item is not found or has expired, the zero value and boolean false are returned.
func (m *MemoryCache[K, V]) Get(key K) (V, bool) {
	// Initialize a variable res to hold the result value, with a zero value for type V.
	// This will be returned if the key is not found or an error occurs.
	var res V

	// Convert the key to a string using fmt.Sprintf for use as a unique identifier.
	// This string is used as the deduplication key in the singleflight group.
	keyStr := fmt.Sprintf("%v", key)

	// Use the singleflight group to deduplicate requests for the same key.
	// If multiple goroutines request the same key simultaneously, only one computation is performed.
	value, err, _ := m.group.Do(keyStr, func() (interface{}, error) {
		// Acquire a read lock to safely access the items map without blocking readers.
		// This ensures thread safety while allowing concurrent reads.
		m.mutex.RLock()
		// Retrieve the linked list element associated with the key from the items map.
		element := m.items[key]
		// Release the read lock to allow other read operations to proceed.
		m.mutex.RUnlock()

		// Check if the element is nil, which means the key does not exist in the cache.
		// If so, return nil, and error to indicate the element is not found.
		if element == nil {
			return res, fmt.Errorf("element is not found")
		}

		// Retrieve the cache item from the linked list element.
		// This step casts the element's value to the expected Item[K, V] type.
		item := element.Value.(*Item[K, V])

		// Check if the item's expiration time is before the current moment.
		// This determines whether the item should be considered expired and removed.
		if item.ExpiresAt.Before(time.Now()) {
			// Acquire a write lock to safely modify the cache state.
			// This ensures no other operations interfere with the removal process.
			m.mutex.Lock()
			// Remove the expired item from the linked list to maintain the LRU order.
			m.list.Remove(element)
			// Delete the expired item's key from the `items` map to release the memory.
			delete(m.items, item.Key)
			// Decrement the size counter to reflect the removal of this item.
			m.size.Add(-1)
			// Release the write lock to allow other operations to proceed.
			m.mutex.Unlock()
			// Return nil to indicate the key is no longer available due to expiration.
			return nil, nil
		}

		// Acquire a write lock to update the recency order of the cache item.
		// Moving the item to the front ensures it is marked as the most recently used.
		m.mutex.Lock()
		// Move the item to the front of the linked list to update its usage.
		m.list.MoveToFront(element)
		// Assign the value of the cache item to res for returning later.
		res = item.Value
		// Release the write lock after updating the recency order.
		m.mutex.Unlock()
		// Return the value of the cache item for use in the singleflight response.
		return res, nil
	})

	// Check if an error occurred during the singleflight execution or if the value is nil.
	// If either condition is true, return the zero value and `false` to indicate failure.
	if err != nil || value == nil {
		return res, false
	}

	// Return the retrieved value cast to type V and true to indicate success.
	// The cast ensures the value conforms to the expected type of the caller.
	return value.(V), true
}

// Contains checks if a given key exists in the cache.
// This method acquires a lock to ensure thread safety while accessing the cache
// and then checks if the key is present in the map of cached items.
func (m *MemoryCache[K, V]) Contains(key K) bool {
	// Acquire a read lock to ensure thread-safe access to the items map.
	// This allows multiple readers to access the cache simultaneously without conflicts.
	m.mutex.RLock()
	// Ensure the read lock is released after the function executes, even in case of errors.
	// Using defer guarantees proper cleanup and prevents potential deadlocks.
	defer m.mutex.RUnlock()

	// Check if the key exists in the items map and store the result in ok.
	// This operation does not modify the cache and is efficient with the read lock held.
	_, ok := m.items[key]

	// Return the boolean value ok indicating whether the key exists in the cache.
	// If ok is true, the key is present; otherwise, it is not.
	return ok
}

// Remove removes an item from the cache by its key.
// This method is thread-safe and ensures that the item is properly removed from both
// the list and the map, as well as from the expiration buckets map.
// It returns true if the item was found and removed, and false if the key does not exist in the cache.
func (m *MemoryCache[K, V]) Remove(key K) bool {
	// Acquire a write lock to ensure thread-safe modification of the cache.
	// This prevents other goroutines from modifying the cache while the removal is in progress.
	m.mutex.Lock()
	// Ensure the lock is released after the function completes, even if an error occurs.
	// Using defer guarantees that the lock will always be unlocked, avoiding potential deadlocks.
	defer m.mutex.Unlock()

	// Check if the key exists in the items map and retrieve its associated list element.
	// If the key is present, the corresponding element will be returned in element.
	if element, ok := m.items[key]; ok {
		// Remove the element from the doubly-linked list (m.list).
		// This operation updates the list structure and adjusts pointers for neighboring elements.
		m.list.Remove(element)
		// Remove the key from the items map to eliminate it from the cache.
		// This ensures that the key is no longer accessible in the cache.
		delete(m.items, key)
		// Decrement the size of the cache to reflect the removal of an item.
		// The cache size is updated atomically for thread-safe size tracking.
		m.size.Add(-1)
		// Note: The removed item remains in the expiration heap.
		// It will be cleaned up later by the collector during heap maintenance.
		return true
	}

	// Return false if the key does not exist in the cache.
	// This indicates that no removal was performed because the key was not found.
	return false
}

// Len retrieves the current number of items in the cache.
// This method provides a thread-safe way to determine the size of the cache by accessing the atomic size counter.
// It ensures that the cache size is accurately reflected, even in concurrent environments.
func (m *MemoryCache[K, V]) Len() int {
	// Access the current size of the cache using the atomic Load method.
	// This operation is thread-safe and ensures consistency across multiple goroutines.
	return int(m.size.Load())
}

// collector runs in the background to periodically remove expired items from the cache.
// It uses a ticker to trigger expiration checks at regular intervals, ensuring the cache stays clean.
// The collector stops running when the parent context signals cancellation, allowing graceful shutdown.
func (m *MemoryCache[K, V]) collector() {
	ticker := time.NewTicker(m.expireCheckInterval)
	defer ticker.Stop()

	// Run an infinite loop that waits for ticker events or cancellation signal.
	// This loop will run until the parent context is canceled, allowing graceful shutdown.
	for {
		select {
		// On every ticker tick, call deleteExpiredData to remove expired items from the cache.
		// This periodically cleans up stale entries without blocking the main cache operations.
		case <-ticker.C:
			m.deleteExpiredData()

		// If the parent context signals done, exit the collector to stop cleanup activities.
		// This ensures that when the cache is no longer needed, the background goroutine terminates properly.
		case <-m.parentCtx.Done():
			return
		}
	}
}

// deleteExpiredData removes all expired items from the cache by checking the expiration heap.
// It ensures that the cache does not hold outdated entries, maintaining freshness and freeing memory.
func (m *MemoryCache[K, V]) deleteExpiredData() {
	// Acquire a write lock to prevent concurrent modifications during expiration cleanup.
	// This guarantees thread safety while removing expired items from multiple data structures.
	m.mutex.Lock()
	// Ensure the lock is released after cleanup is done to allow other operations to proceed.
	defer m.mutex.Unlock()

	// Capture the current time to compare with item expiration times.
	// This is used to determine which items are expired and should be removed.
	now := time.Now()

	// Iterate through the expiration heap while there are items present.
	// The heap is ordered by expiration time, so expired items will be at the front.
	for m.expirationHeap.Len() > 0 {
		// Peek at the item with the earliest expiration time.
		item := m.expirationHeap[0]
		// If the earliest item has not yet expired, stop processing further items.
		// Because heap is ordered, no subsequent items can be expired.
		if item.ExpiresAt.After(now) {
			break
		}

		// Remove the expired item from the expiration heap.
		heap.Pop(&m.expirationHeap)
		// Check if the expired item is still present in the cache's items map.
		if element, ok := m.items[item.Key]; ok {
			// Remove the expired item from the linked list used for LRU tracking.
			m.list.Remove(element)
			// Delete the expired item's entry from the items map.
			delete(m.items, item.Key)
			// Decrement the size counter as one item is removed from the cache.
			m.size.Add(-1)
		}
	}
}
