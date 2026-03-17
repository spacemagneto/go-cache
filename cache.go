package go_cache

import "time"

// Cache defines the interface for a generic in-memory cache with TTL support.
// The interface is parameterized by types K and V, where K is the type of the keys and V is the type of the values.
// This interface supports basic cache operations such as setting, getting, checking, and removing items, as well as retrieving the cache length.
type Cache[K comparable, V any] interface {
	// Set adds or updates an item in the cache with the specified key, value, and TTL (time-to-live).
	// If the TTL is zero, the default TTL is used. This method is thread-safe and ensures
	// that the cache is updated atomically.
	// key: The key under which the value is stored.
	// value: The value to store in the cache.
	// ttl: The time-to-live duration for the cache item. After this duration, the item will be expired.
	Set(key K, value V, ttl time.Duration)

	// Contains checks if the cache contains an item with the specified key.
	// Returns true if the item exists in the cache and has not expired, otherwise false.
	// key: The key to check in the cache.
	Contains(key K) bool

	// Get retrieves an item from the cache by its key.
	// If the item is found and has not expired, it is returned along with a boolean true.
	// If the item is not found or has expired, the zero value and boolean false are returned.
	// key: The key of the item to retrieve.
	// returns: The value associated with the key and a boolean indicating if the item was found and is not expired.
	Get(key K) (item V, ok bool)

	// Remove deletes an item from the cache by its key.
	// This method is thread-safe and ensures that the item is properly removed from both
	// the list and the map, as well as from the expiration buckets map.
	// Returns true if the item was found and removed, and false if the key does not exist in the cache.
	// key: The key of the item to remove.
	// returns: A boolean indicating if the item was found and removed.
	Remove(key K) bool

	// Len returns the number of items currently stored in the cache.
	// This method provides the current count of items, including those that may be expired but not yet removed.
	// returns: The number of items in the cache.
	Len() int

	// Close stops the background collector and clears all internal data structures.
	// After calling Close, all other methods return zero values or false immediately.
	Close()
}
