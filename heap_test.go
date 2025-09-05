package go_cache

import (
	"container/heap"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestExpirationHeap(t *testing.T) {
	t.Parallel()

	// Test pushing and popping multiple items to verify heap ordering.
	// This sub-test ensures that items are popped in order of earliest expiration.
	t.Run("PushAndPopMultipleItems", func(t *testing.T) {
		// Create a new ExpirationHeap for string keys and string values.
		// The heap will store items ordered by their ExpiresAt timestamps.
		expirationHeap := &ExpirationHeap[string, string]{}
		// Initialize the heap to satisfy container/heap.Interface.
		// This ensures heap operations like Push and Pop maintain the min-heap property.
		heap.Init(expirationHeap)
		// Capture the current time to use as a base for expiration timestamps.
		// This provides a consistent starting point for ExpiresAt values.
		now := time.Now()
		// Define the first item with a 3-second expiration time.
		// This item will expire last among the test items.
		itemFirst := &Item[string, string]{Key: "key1", Value: "value1", ExpiresAt: now.Add(3 * time.Second)}
		// Define the second item with a 1-second expiration time.
		// This item will expire first among the test items.
		itemSecond := &Item[string, string]{Key: "key2", Value: "value2", ExpiresAt: now.Add(1 * time.Second)}
		// Define the third item with a 2-second expiration time.
		// This item will expire second among the test items.
		itemThird := &Item[string, string]{Key: "key3", Value: "value3", ExpiresAt: now.Add(2 * time.Second)}

		// Push the first item onto the heap.
		// This adds itemFirst to the heap, maintaining the min-heap property.
		heap.Push(expirationHeap, itemFirst)
		// Push the second item onto the heap.
		// This adds itemSecond, which should become the root due to its earlier expiration.
		heap.Push(expirationHeap, itemSecond)
		// Push the third item onto the heap.
		// This adds itemThird, adjusting the heap to keep itemSecond as the root.
		heap.Push(expirationHeap, itemThird)
		// Verify the heap's length after pushing three items.
		// The length should be 3, reflecting the number of items added.
		assert.Equal(t, 3, expirationHeap.Len(), "Expected heap length to be 3 after pushing three items")
		// Pop the first item from the heap.
		// This should return itemSecond (key2), as it has the earliest expiration (1s).
		popped1 := heap.Pop(expirationHeap).(*Item[string, string])
		// Assert that the first popped item's key is key2.
		// This confirms the min-heap property prioritizes the earliest expiration.
		assert.Equal(t, "key2", popped1.Key, "Expected first popped item to have key key2")
		// Verify the heap's length after the first pop.
		// The length should be 2, as one item was removed.
		assert.Equal(t, 2, expirationHeap.Len(), "Expected heap length to be 2 after first pop")
		// Pop the second item from the heap.
		// This should return itemThird (key3), as it has the next earliest expiration (2s).
		popped2 := heap.Pop(expirationHeap).(*Item[string, string])
		// Assert that the second popped item's key is key3.
		// This confirms the heap reorders correctly after the first pop.
		assert.Equal(t, "key3", popped2.Key, "Expected second popped item to have key key3")
		// Verify the heap's length after the second pop.
		// The length should be 1, as two items were removed.
		assert.Equal(t, 1, expirationHeap.Len(), "Expected heap length to be 1 after second pop")
		// Pop the third item from the heap.
		// This should return itemFirst (key1), as it has the last expiration (3s).
		popped3 := heap.Pop(expirationHeap).(*Item[string, string])
		// Assert that the third popped item's key is key1.
		// This confirms all items were popped in correct order.
		assert.Equal(t, "key1", popped3.Key, "Expected third popped item to have key key1")
		// Verify the heap's length after the third pop.
		// The length should be 0, as all items were removed.
		assert.Equal(t, 0, expirationHeap.Len(), "Expected heap length to be 0 after third pop")
	})

	// Test pushing and popping a single item to verify basic functionality.
	// This sub-test ensures the heap handles a single item correctly.
	t.Run("SingleItem", func(t *testing.T) {
		// Create a new ExpirationHeap for string keys and string values.
		// This heap will be used to test single-item operations.
		expirationHeap := &ExpirationHeap[string, string]{}
		// Initialize the heap to satisfy container/heap.Interface.
		// This prepares the heap for Push and Pop operations.
		heap.Init(expirationHeap)
		// Capture the current time to set the item's expiration.
		// This provides a timestamp for the ExpiresAt field.
		now := time.Now()
		// Define a single item with a 1-second expiration time.
		// This item will be pushed and popped to test heap behavior.
		item := &Item[string, string]{Key: "key1", Value: "value1", ExpiresAt: now.Add(1 * time.Second)}

		// Push the item onto the heap.
		// This adds the item as the root of the empty heap.
		heap.Push(expirationHeap, item)
		// Verify the heap's length after pushing one item.
		// The length should be 1, reflecting the single item added.
		assert.Equal(t, 1, expirationHeap.Len(), "Expected heap length to be 1 after pushing one item")
		// Pop the item from the heap.
		// This should return the single item (key1).
		popped := heap.Pop(expirationHeap).(*Item[string, string])
		// Assert that the popped item's key is key1.
		// This confirms the item was correctly stored and retrieved.
		assert.Equal(t, "key1", popped.Key, "Expected popped item to have key key1")
		// Assert that the popped item's value is value1.
		// This verifies the item's value was preserved.
		assert.Equal(t, "value1", popped.Value, "Expected popped item to have value value1")
		// Assert that the popped item's ExpiresAt matches the original.
		// This ensures the expiration time was correctly maintained.
		assert.Equal(t, item.ExpiresAt, popped.ExpiresAt, "Expected popped item to have matching ExpiresAt")
		// Verify the heap's length after popping.
		// The length should be 0, as the only item was removed.
		assert.Equal(t, 0, expirationHeap.Len(), "Expected heap length to be 0 after popping single item")
	})

	// Test operations on an empty heap to verify edge case behavior.
	// This sub-test ensures Len returns 0 and Pop is handled safely.
	t.Run("EmptyHeap", func(t *testing.T) {
		// Create a new ExpirationHeap for string keys and string values.
		// This heap will be tested in its empty state.
		expirationHeap := &ExpirationHeap[string, string]{}
		// Initialize the heap to satisfy container/heap.Interface.
		// This ensures the heap is in a valid state for operations.
		heap.Init(expirationHeap)
		// Verify the length of the empty heap.
		// The length should be 0, as no items have been added.
		assert.Equal(t, 0, expirationHeap.Len(), "Expected empty heap length to be 0")
	})

	// Test the Less method to verify correct ordering by ExpiresAt.
	// This sub-test ensures the min-heap property is maintained.
	t.Run("LessOrdering", func(t *testing.T) {
		// Create a new ExpirationHeap with two pre-populated items.
		// The items have different expiration times to test Less.
		now := time.Now()

		expirationHeap := &ExpirationHeap[string, string]{
			// Add an item with a 2-second expiration time.
			// This item expires later than the second item.
			&Item[string, string]{Key: "key1", ExpiresAt: now.Add(2 * time.Second)},
			// Add an item with a 1-second expiration time.
			// This item expires earlier than the first item.
			&Item[string, string]{Key: "key2", ExpiresAt: now.Add(1 * time.Second)},
		}

		// Verify that Less(1, 0) returns true.
		// This confirms that the earlier expiration (1s) is prioritized over the later (2s).
		assert.True(t, expirationHeap.Less(1, 0), "Expected Less(1, 0) to return true for 1s < 2s")
		// Verify that Less(0, 1) returns false.
		// This confirms that the later expiration (2s) is not prioritized over the earlier (1s).
		assert.False(t, expirationHeap.Less(0, 1), "Expected Less(0, 1) to return false for 2s > 1s")
	})

	// Test the Swap method to verify correct element exchange.
	// This sub-test ensures Swap correctly reorders items.
	t.Run("Swap", func(t *testing.T) {
		// Create a new ExpirationHeap with two items.
		// The items have different keys and values to test swapping.
		now := time.Now()

		expirationHeap := &ExpirationHeap[string, string]{
			// Add an item with key1 and a 1-second expiration.
			// This item will be swapped to index 1.
			&Item[string, string]{Key: "key1", Value: "value1", ExpiresAt: now.Add(1 * time.Second)},
			// Add an item with key2 and a 2-second expiration.
			// This item will be swapped to index 0.
			&Item[string, string]{Key: "key2", Value: "value2", ExpiresAt: now.Add(2 * time.Second)},
		}
		// Swap the items at indices 0 and 1.
		// This exchanges their positions in the heap.
		expirationHeap.Swap(0, 1)
		// Verify that index 0 now contains key2.
		// This confirms the swap moved key2 to the first position.
		assert.Equal(t, "key2", (*expirationHeap)[0].Key, "Expected index 0 to have key key2 after swap")
		// Verify that index 0 has value2.
		// This confirms the swap preserved the value of key2.
		assert.Equal(t, "value2", (*expirationHeap)[0].Value, "Expected index 0 to have value value2 after swap")
		// Verify that index 1 now contains key1.
		// This confirms the swap moved key1 to the second position.
		assert.Equal(t, "key1", (*expirationHeap)[1].Key, "Expected index 1 to have key key1 after swap")
		// Verify that index 1 has value1.
		// This confirms the swap preserved the value of key1.
		assert.Equal(t, "value1", (*expirationHeap)[1].Value, "Expected index 1 to have value value1 after swap")
	})

	// Test a large heap with many items to stress the heap operations.
	// This sub-test verifies that the heap maintains order with a large dataset.
	t.Run("LargeHeap", func(t *testing.T) {
		// Create a new ExpirationHeap for string keys and string values.
		// This heap will be used to test large-scale operations.
		expirationHeap := &ExpirationHeap[string, string]{}
		// Initialize the heap to satisfy container/heap.Interface.
		// This prepares the heap for pushing and popping many items.
		heap.Init(expirationHeap)
		// Capture the current time to set expiration timestamps.
		// This provides a base for generating unique ExpiresAt values.
		now := time.Now()
		// Define the number of items to push onto the heap.
		// 1000 items will stress the heap's performance and ordering.
		const n = 1000
		// Create a slice to store references to the pushed items.
		// This allows tracking the items for later verification.
		items := make([]*Item[string, string], n)
		// Populate the heap with 1000 items, each with a unique expiration time.
		// Expiration times increase by 1ms per item to ensure distinct ordering.
		for i := 0; i < n; i++ {
			// Create a new item with a unique key and expiration time.
			// The expiration time is set to now + i milliseconds.
			items[i] = &Item[string, string]{Key: fmt.Sprintf("key%d", i), Value: fmt.Sprintf("value%d", i), ExpiresAt: now.Add(time.Duration(i) * time.Millisecond)}

			// Push the item onto the heap.
			// This adds the item while maintaining the min-heap property.
			heap.Push(expirationHeap, items[i])
		}

		// Verify the heap's length after pushing 1000 items.
		// The length should be 1000, reflecting all added items.
		assert.Equal(t, n, expirationHeap.Len(), "Expected heap length to be 1000 after pushing 1000 items")
		// Initialize a variable to track the last popped item's expiration time.
		// This will be used to ensure items are popped in ascending order.
		var last time.Time
		// Pop all items from the heap and verify their order.
		// Items should be popped in ascending order of ExpiresAt.
		for i := 0; i < n; i++ {
			// Pop the next item from the heap.
			// This should return the item with the earliest remaining expiration.
			item := heap.Pop(expirationHeap).(*Item[string, string])
			// If this is not the first item, verify that its expiration time is not before the last.
			// This ensures the min-heap property is maintained (ascending order).
			if i > 0 {
				assert.False(t, item.ExpiresAt.Before(last), "Expected item %d to not expire before previous item", i)
			}
			// Update the last expiration time for the next iteration.
			// This tracks the most recently popped item's ExpiresAt.
			last = item.ExpiresAt
			// Verify the heap's length after the current pop.
			// The length should decrease by 1 for each pop.
			assert.Equal(t, n-1-i, expirationHeap.Len(), "Expected heap length to be %d after pop %d", n-1-i, i+1)
		}

		// Verify the heap's length after all pops.
		// The length should be 0, as all items were removed.
		assert.Equal(t, 0, expirationHeap.Len(), "Expected heap length to be 0 after popping all items")
	})
}
