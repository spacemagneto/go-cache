# Go Cache

![Coverage](https://img.shields.io/badge/Coverage-100%25-brightgreen.svg)

go-cache is a generic, thread-safe, in-memory cache package for Go (1.18+). It supports time-to-live (TTL) expiration, least-recently-used (LRU) eviction, and cache stampede prevention using the singleflight package. The cache is designed for high performance and reliability, with atomic operations for size tracking and a background goroutine for periodic cleanup of expired items.

---------

### Installation

```go
go get github.com/spacemagneto/go-cache
```


## Features

- **Generic Cache:** Supports any comparable key type and any value type using Go generics.
- **TTL Expiration:** Items expire after a configurable time-to-live duration.
- **LRU Eviction:** Evicts least recently used items when the cache reaches its maximum size.
- **Cache Stampede Prevention:** Uses singleflight to deduplicate concurrent requests for the same key.
- **Thread-Safe:** Utilizes sync.RWMutex and atomic operations for safe concurrent access.
- **Background Cleanup:** Periodically removes expired items via a background goroutine.
- **Configurable:** Customizable TTL, maximum items, and expiration check interval.


## Usage Examples

### Setting and Getting Items

Add an item to the cache and retrieve it.


```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/spacemagneto/go-cache"
)

func main() {
    ctx := context.Background()
    cache := go_cache.NewMemoryCache[string, int](ctx, time.Hour, 10*time.Second, 100)

    // Set an item
    cache.Set("key1", 42, 5*time.Minute)

    // Get the item
    if value, ok := cache.Get("key1"); ok {
        fmt.Printf("Found value: %d\n", value) // Output: Found value: 42
    } else {
        fmt.Println("Key not found or expired")
    }
}
```

### Checking for Key Existence

Check if a key exists in the cache.

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/spacemagneto/go-cache"
)

func main() {
    ctx := context.Background()
    cache := go_cache.NewMemoryCache[string, int](ctx, time.Hour, 10*time.Second, 100)

    cache.Set("key1", 42, 5*time.Minute)
    exists := cache.Contains("key1")
    fmt.Println(exists) // Output: true

    exists = cache.Contains("key2")
    fmt.Println(exists) // Output: false
}
```

### Removing an Item

Remove an item from the cache by its key.

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/spacemagneto/go-cache"
)

func main() {
    ctx := context.Background()
    cache := go_cache.NewMemoryCache[string, int](ctx, time.Hour, 10*time.Second, 100)

    cache.Set("key1", 42, 5*time.Minute)
    removed := cache.Remove("key1")
    fmt.Println(removed) // Output: true

    exists := cache.Contains("key1")
    fmt.Println(exists) // Output: false
}
```

### Checking Cache Size

Get the current number of items in the cache.

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/spacemagneto/go-cache"
)

func main() {
    ctx := context.Background()
    cache := go_cache.NewMemoryCache[string, int](ctx, time.Hour, 10*time.Second, 100)

    cache.Set("key1", 42, 5*time.Minute)
    cache.Set("key2", 100, 5*time.Minute)
    size := cache.Len()
    fmt.Printf("Cache size: %d\n", size) // Output: Cache size: 2
}
```

# License

This package is licensed under the Apache License, Version 2.0. See the LICENSE file for details.