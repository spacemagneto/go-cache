package go_cache

// ExpirationHeap implements a min-heap for items based on expiration time.
type ExpirationHeap[K comparable, V any] []*Item[K, V]

// Len returns the number of elements currently stored in the ExpirationHeap.
// It provides a quick way to determine the size of the heap without modifying its contents.
// This method is essential for heap operations where the size determines validity or termination conditions.
func (h ExpirationHeap[K, V]) Len() int {
	// Use the built-in len function to calculate and return the number of elements in the heap.
	// This leverages the slice backing of the heap for efficient size retrieval.
	return len(h)
}

// Less determines the relative order of two elements in the ExpirationHeap.
// It is used by the heap interface to maintain the heap property during operations like insertions and deletions.
// This method compares the ExpiresAt timestamps of two elements and returns true if the element at index i
// expires earlier than the element at index j, ensuring that the heap is sorted by expiration time.
func (h ExpirationHeap[K, V]) Less(i, j int) bool {
	// Compare the ExpiresAt timestamps of the elements at indices i and j.
	// Returns true if the element at index i expires earlier than the one at index j.
	return h[i].ExpiresAt.Before(h[j].ExpiresAt)
}

// Swap exchanges the positions of two elements in the ExpirationHeap.
// This method is used by the heap interface during operations such as reordering
// after insertions or deletions. By swapping the elements at indices i and j,
// the method helps maintain the heap structure and ensures correctness during heap operations.
func (h ExpirationHeap[K, V]) Swap(i, j int) {
	// Exchange the elements at indices i and j in the underlying slice.
	// This reorders the heap elements to maintain the required heap property.
	h[i], h[j] = h[j], h[i]
}

// Push adds a new item to the ExpirationHeap.
// This method is used by the heap interface to insert a new element into the heap.
// The new element is appended to the underlying slice, and the heap property is
// maintained through subsequent heapify operations.
func (h *ExpirationHeap[K, V]) Push(x interface{}) {
	// Append the new item to the slice representing the heap.
	// The item is type-asserted to *Item[K, V] to ensure it matches the expected type.
	*h = append(*h, x.(*Item[K, V]))
}

// Pop removes and returns the last item from the ExpirationHeap.
// This method is used by the heap interface to remove the element with the highest index.
// It decreases the size of the heap and ensures the heap property is preserved for subsequent operations.
func (h *ExpirationHeap[K, V]) Pop() interface{} {
	// Store the current heap slice to manipulate its elements.
	old := *h
	// Determine the current length of the heap slice.
	n := len(old)
	// Retrieve the last element in the heap, which is the item to be removed.
	// This is necessary to return the removed item after resizing the heap.
	x := old[n-1]
	// Resize the heap slice by excluding the last element, effectively removing it.
	*h = old[0 : n-1]

	// Return the removed item as the result of the operation.
	return x
}
