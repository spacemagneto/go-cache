# Go Cache

![Coverage](https://img.shields.io/badge/Coverage-100%25-brightgreen.svg)
![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8.svg)
![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)

go-cache is a generic, thread-safe, in-memory cache for Go 1.21+. It supports per-item TTL expiration and LRU eviction, and is designed for high read throughput: expired items discovered during `Get` or `Contains` are never removed on the hot path — they are handed off to a background collector, which means reads never take a write lock for cleanup.

---------

## Installation

```sh
go get github.com/spacemagneto/go-cache
```

## Quick start

```go
package main
 
import (
    "context"
    "fmt"
    "time"
 
    gocache "github.com/spacemagneto/go-cache"
)
 
func main() {
    cache := gocache.NewMemoryCache[string, int](
        context.Background(),
        time.Hour,       // default TTL
        10*time.Minute,  // expiration check interval
        10_000,          // max items
    )
    defer cache.Close()
 
    cache.Set("hits", 42, 5*time.Minute)
 
    if value, ok := cache.Get("hits"); ok {
        fmt.Println(value) // 42
    }
}
```

## Creating a cache

```go
cache := gocache.NewMemoryCache[K, V](ctx, ttl, expireCheckInterval, maxItems)
```

| Parameter           | Type              | Default        | Description                                                                          |
|---------------------|-------------------|----------------|--------------------------------------------------------------------------------------|
| `ctx`               | `context.Context` | —              | Parent context. Cancelling it is equivalent to calling `Close`.                     |
| `ttl`               | `time.Duration`   | `1h`           | Default item lifetime. Used when `Set` is called with `ttl = 0`.                    |
| `expireCheckInterval` | `time.Duration` | `10m`          | How often the background collector sweeps for expired items.                         |
| `maxItems`          | `int`             | `10 000`       | Capacity limit. When reached, the least-recently-used item is evicted on each `Set`. |

All zero values fall back to the defaults above, so the minimal valid call is:

```go
cache := gocache.NewMemoryCache[string, int](context.Background(), 0, 0, 0)
```

## API

### Set

```go
cache.Set(key K, value V, ttl time.Duration)
```

Stores or overwrites `key` with `value`. Pass `ttl = 0` to use the cache default. If the cache is at capacity the least-recently-used item is evicted before the new item is inserted. Re-setting an existing key replaces the value and resets the TTL; the previous heap entry is invalidated by a version bump and discarded lazily by the collector.

Calling `Set` on a closed cache is a no-op.

### Get

```go
value, ok := cache.Get(key K)
```

Returns the value and `true` when the key exists and has not expired. Returns the zero value and `false` otherwise. On a cache hit the item is moved to the most-recently-used position. An expired item discovered during `Get` is forwarded to the collector via a non-blocking channel; the write lock is never taken on the hot path.

### Contains

```go
ok := cache.Contains(key K)
```

Reports whether the key exists and has not expired, without updating the LRU order. Expired items are forwarded to the collector the same way as in `Get`.

### Remove

```go
ok := cache.Remove(key K)
```

Deletes the key and returns `true` if it existed. The corresponding heap entry is not removed immediately; it becomes a tombstone that the collector discards on the next sweep using a version check.

### Len

```go
n := cache.Len()
```

Returns the number of live items in the cache. The counter is maintained with `atomic.Int32` and never requires locking.

### Close

```go
cache.Close()
```

Stops the background collector, waits for it to exit, and releases internal data structures. After `Close` returns, all methods return their zero/false values immediately. `Close` is idempotent.

## Expiration model

Expiration works through two complementary paths that together avoid write locks on the hot read path:

**Hot path (Get / Contains).** When a read discovers an expired item it does not delete it inline. Instead it sends the key to an internal buffered channel (`pendingDeleteCh`) using a non-blocking `select`, then returns immediately. The write lock is never taken.

**Background collector.** A single goroutine runs on a configurable tick interval. On each tick it first drains `pendingDeleteCh` (removing the keys flagged by `Get` and `Contains`), then walks a min-heap ordered by `ExpiresAt` to remove any remaining elapsed entries. Because the heap is sorted, the walk stops as soon as it reaches the first non-expired item — the full heap is never scanned.

**Lazy heap deletion.** `Remove` and LRU eviction delete an item from the `items` map and the LRU list immediately, but intentionally leave the corresponding heap entry in place. That entry is called a tombstone. When the collector pops it during the next sweep, it compares the entry's version number against the version of the live item in the map. A mismatch (or an absent key) means the entry is stale and it is silently discarded, with no further effect on the map.

This design means that every `Remove` or LRU eviction is O(1), and the heap is kept clean on a background schedule rather than on the write path.

## LRU eviction

When `maxItems` is reached, `Set` evicts the least-recently-used item before inserting the new one. LRU order is maintained via a doubly-linked list: `Set` and `Get` move the accessed item to the front; the eviction candidate is always the element at the back. `Contains` deliberately does not update LRU order.

## Defaults

| Constant                  | Value    |
|---------------------------|----------|
| `DefaultTTL`              | `1h`     |
| `DefaultExpireCheckInterval` | `10m` |
| `DefaultMaxItems`         | `10 000` |

## Usage examples

### TTL per item

```go
cache.Set("session:abc", sessionData, 30*time.Minute)
cache.Set("config:flags", flags, 24*time.Hour)
cache.Set("rate:user:42", counter, time.Minute)
```

### Check before read

```go
if cache.Contains("report:2024-Q4") {
    value, _ := cache.Get("report:2024-Q4")
    render(value)
}
```

`Contains` does not promote the item in LRU order, so it is appropriate for existence checks that should not influence eviction priority.

### Graceful shutdown

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
defer stop()
 
cache := gocache.NewMemoryCache[string, []byte](ctx, time.Hour, 10*time.Minute, 50_000)
 
// When the process receives SIGINT, ctx is cancelled, which stops the
// collector automatically. Alternatively, call cache.Close() directly.
<-ctx.Done()
cache.Close()
```

### Using the Cache interface

`MemoryCache` implements the `Cache[K, V]` interface, which makes it straightforward to swap implementations in tests or behind a dependency-injection boundary:

```go
type Service struct {
    cache gocache.Cache[string, User]
}
 
func NewService(cache gocache.Cache[string, User]) *Service {
    return &Service{cache: cache}
}
```

## Performance notes

`Get` and `Contains` acquire a read lock for the lookup and release it before returning. A write lock is taken only to update LRU order on a confirmed cache hit. Reads of expired items send a key to `pendingDeleteCh` with a non-blocking send, so they never stall even if the channel is full — the collector will clean up those items on the next tick.

The `size` counter uses `atomic.Int32`, so `Len` never acquires any lock.

## License

This package is licensed under the Apache License, Version 2.0. See the LICENSE file for details.