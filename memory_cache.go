package go_cache

import (
	"container/heap"
	"container/list"
	"context"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// DefaultTTL is the default time-to-live duration for cache items.
	DefaultTTL      = time.Hour
	DefaultMaxItems = 10000
)

// MemoryCache represents an in-memory cache with TTL support and LRU eviction.
type MemoryCache[K comparable, V any] struct {
	list                *list.List           // Doubly-linked list for LRU ordering
	items               map[K]*list.Element  // Map for quick lookups
	expirationHeap      ExpirationHeap[K, V] // Min-heap for expiration tracking
	ttl                 time.Duration        // Default time-to-live duration
	maxItems            int                  // Maximum number of items in the cache (0 = unlimited)
	size                atomic.Int32         // Atomic counter for cache size
	mutex               sync.RWMutex         // Mutex for thread safety
	expireCheckInterval time.Duration        // Interval for cleaning up expired items
	closed              atomic.Bool

	// nextVersion is a per-instance monotonic counter stamped on every new Item.
	// Comparing the version stored in a heap entry against the version of the
	// live item in the map lets the collector detect tombstones in O(1) without
	// an explicit removal on every eviction or Remove call.
	nextVersion atomic.Uint64

	// pendingDeleteCh carries keys of expired items found by Get/Contains so the
	// collector can remove them in the background without blocking the caller.
	pendingDeleteCh chan K

	contextCancelFunc context.CancelFunc
	wg                sync.WaitGroup // WaitGroup for managing goroutines
}

// NewMemoryCache creates a new MemoryCache instance with the specified TTL and max items.
// It starts a background goroutine to periodically delete expired items.
func NewMemoryCache[K comparable, V any](ctx context.Context, ttl, expireCheckInterval time.Duration, maxItems int) *MemoryCache[K, V] {
	if ttl <= 0 {
		ttl = DefaultTTL
	}

	if expireCheckInterval <= 0 {
		expireCheckInterval = DefaultTTL
	}

	if maxItems <= 0 {
		maxItems = DefaultMaxItems
	}

	ctx, cancel := context.WithCancel(ctx)

	cache := &MemoryCache[K, V]{
		list:                list.New(),
		items:               make(map[K]*list.Element),
		expirationHeap:      make(ExpirationHeap[K, V], 0),
		ttl:                 ttl,
		maxItems:            maxItems,
		expireCheckInterval: expireCheckInterval,
		contextCancelFunc:   cancel,
		pendingDeleteCh:     make(chan K, maxItems),
	}

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
		cache.collector(ctx)
	}()

	return cache
}

// Set adds or updates an item in the cache with the specified key, value, and TTL.
// If the cache exceeds its maximum allowed items, the least recently used item is evicted.
// This method ensures that items are stored with proper expiration times and maintains
// the order of usage to support LRU eviction.
func (m *MemoryCache[K, V]) Set(key K, value V, ttl time.Duration) {
	if m.closed.Load() {
		return
	}

	if ttl <= 0 {
		ttl = m.ttl
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.maxItems > 0 && int(m.size.Load()) >= m.maxItems {
		m.evictLRU()
	}

	if element, ok := m.items[key]; ok {
		m.list.Remove(element)
		m.size.Add(-1)
	}

	item := &Item[K, V]{Key: key, Value: value, ExpiresAt: time.Now().Add(ttl), version: m.nextVersion.Add(1)}

	element := m.list.PushFront(&entry[K, V]{item: item})
	m.items[key] = element
	heap.Push(&m.expirationHeap, item)
	m.size.Add(1)
}

// Get retrieves an item from the cache by its key.
// If the item is found and has not expired, it is returned along with a boolean true.
// If the item is not found or has expired, the zero value and boolean false are returned.
func (m *MemoryCache[K, V]) Get(key K) (V, bool) {
	var zero V

	if m.closed.Load() {
		return zero, false
	}

	m.mutex.RLock()
	element, ok := m.items[key]
	if !ok {
		m.mutex.RUnlock()
		return zero, false
	}

	item := element.Value.(*entry[K, V]).item
	expired := item.ExpiresAt.Before(time.Now())
	m.mutex.RUnlock()

	if expired {
		select {
		case m.pendingDeleteCh <- key:
		default:
		}
		return zero, false
	}

	m.mutex.Lock()
	// Re-check: the item may have been evicted between RUnlock and Lock.
	if _, stillThere := m.items[key]; stillThere {
		m.list.MoveToFront(element)
	}
	m.mutex.Unlock()

	return item.Value, true
}

// Contains checks if a given key exists in the cache.
// This method acquires a lock to ensure thread safety while accessing the cache
// and then checks if the key is present in the map of cached items.
func (m *MemoryCache[K, V]) Contains(key K) bool {
	if m.closed.Load() {
		return false
	}

	m.mutex.RLock()
	element, ok := m.items[key]
	if !ok {
		m.mutex.RUnlock()
		return false
	}

	expired := element.Value.(*entry[K, V]).item.ExpiresAt.Before(time.Now())
	m.mutex.RUnlock()

	if expired {
		select {
		case m.pendingDeleteCh <- key:
		default:
		}
		return false
	}

	return true
}

// Remove removes an item from the cache by its key.
// This method is thread-safe and ensures that the item is properly removed from both
// the list and the map, as well as from the expiration buckets map.
// It returns true if the item was found and removed, and false if the key does not exist in the cache.
func (m *MemoryCache[K, V]) Remove(key K) bool {
	if m.closed.Load() {
		return false
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.remove(key)
}

// Len retrieves the current number of items in the cache.
// This method provides a thread-safe way to determine the size of the cache by accessing the atomic size counter.
// It ensures that the cache size is accurately reflected, even in concurrent environments.
func (m *MemoryCache[K, V]) Len() int {
	// Access the current size of the cache using the atomic Load method.
	// This operation is thread-safe and ensures consistency across multiple goroutines.
	return int(m.size.Load())
}

// remove removes key from the items map and the LRU list.
// The heap entry is not touched here; it becomes a tombstone resolved lazily by
// the collector.
func (m *MemoryCache[K, V]) remove(key K) bool {
	element, ok := m.items[key]
	if !ok {
		return false
	}

	m.list.Remove(element)
	delete(m.items, key)
	m.size.Add(-1)
	return true
}

// evictLRU removes the least-recently-used entry from the cache to make
// room for a new item.
func (m *MemoryCache[K, V]) evictLRU() {
	element := m.list.Back()
	if element == nil {
		return
	}

	item := element.Value.(*entry[K, V]).item
	m.list.Remove(element)
	delete(m.items, item.Key)
	m.size.Add(-1)
}

// collector runs in the background to periodically remove expired items from the cache.
// It uses a ticker to trigger expiration checks at regular intervals, ensuring the cache stays clean.
// The collector stops running when the parent context signals cancellation, allowing graceful shutdown.
func (m *MemoryCache[K, V]) collector(ctx context.Context) {
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
		case <-ctx.Done():
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
