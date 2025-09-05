package go_cache

import "time"

// Item represents a generic item that can be stored in a priority queue.
// It contains a key of type Key, a value of type Value, an expiration time,
// and an index for heap management.
type Item[Key comparable, Value any] struct {
	Key       Key       // Key is a unique identifier for the item.
	Value     Value     // Value is the data associated with the item.
	ExpiresAt time.Time // ExpiresAt is the time at which the item is considered expired.
}
