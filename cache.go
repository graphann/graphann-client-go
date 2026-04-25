package graphann

import (
	"container/list"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"
)

// queryCache is an LRU + TTL cache of SearchResponse keyed on a stable
// fingerprint of the request. nil-receiver methods are safe and act as
// a no-op cache (Get returns false, Set does nothing) so call sites
// don't need to nil-check.
type queryCache struct {
	mu      sync.Mutex
	max     int
	ttl     time.Duration
	entries map[string]*list.Element
	order   *list.List
}

type cacheEntry struct {
	key    string
	value  *SearchResponse
	expiry time.Time
}

// newQueryCache returns a queryCache with the given limits, or nil if
// the cache is effectively disabled (max <= 0 or ttl <= 0).
func newQueryCache(max int, ttl time.Duration) *queryCache {
	if max <= 0 || ttl <= 0 {
		return nil
	}
	return &queryCache{
		max:     max,
		ttl:     ttl,
		entries: make(map[string]*list.Element, max),
		order:   list.New(),
	}
}

// Get returns the cached value if present and not expired.
func (c *queryCache) Get(key string) (*SearchResponse, bool) {
	if c == nil {
		return nil, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	entry := el.Value.(*cacheEntry)
	if time.Now().After(entry.expiry) {
		c.order.Remove(el)
		delete(c.entries, key)
		return nil, false
	}
	c.order.MoveToFront(el)
	return entry.value, true
}

// Set stores the value under key. Evicts the oldest entry when at
// capacity.
func (c *queryCache) Set(key string, value *SearchResponse) {
	if c == nil || value == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.entries[key]; ok {
		entry := el.Value.(*cacheEntry)
		entry.value = value
		entry.expiry = time.Now().Add(c.ttl)
		c.order.MoveToFront(el)
		return
	}
	if c.order.Len() >= c.max {
		oldest := c.order.Back()
		if oldest != nil {
			c.order.Remove(oldest)
			delete(c.entries, oldest.Value.(*cacheEntry).key)
		}
	}
	el := c.order.PushFront(&cacheEntry{
		key:    key,
		value:  value,
		expiry: time.Now().Add(c.ttl),
	})
	c.entries[key] = el
}

// Len returns the number of entries currently cached. Safe on nil
// receiver.
func (c *queryCache) Len() int {
	if c == nil {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.order.Len()
}

// Purge removes all entries from the cache. Safe on nil receiver.
func (c *queryCache) Purge() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*list.Element, c.max)
	c.order.Init()
}

// fingerprint returns a stable hex-encoded SHA-256 of the marshalled
// args. Used as the cache + singleflight key. Marshalling failures are
// reported by returning empty string (callers must skip cache /
// singleflight in that case).
func fingerprint(parts ...any) string {
	var buf []byte
	enc := sha256.New()
	for _, p := range parts {
		b, err := json.Marshal(p)
		if err != nil {
			return ""
		}
		buf = append(buf[:0], b...)
		_, _ = enc.Write(buf)
		_, _ = enc.Write([]byte{0x1e}) // RS separator
	}
	return hex.EncodeToString(enc.Sum(nil))
}
