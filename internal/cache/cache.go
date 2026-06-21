// Package cache implements a bounded concurrent in-memory response cache.
package cache

import (
	"container/list"
	"net/http"
	"sync"
	"time"
)

// Response is an immutable cached HTTP response snapshot.
type Response struct {
	Status int
	Header http.Header
	Body   []byte
	Stored time.Time
}

type item struct {
	key       string
	response  Response
	expiresAt time.Time
	size      int64
}

// Cache is a least-recently-used cache bounded by entry count and bytes.
type Cache struct {
	mu         sync.Mutex
	items      map[string]*list.Element
	lru        *list.List
	maxEntries int
	maxBytes   int64
	bytes      int64
	now        func() time.Time
}

// New creates an empty bounded cache.
func New(maxEntries int, maxBytes int64) *Cache {
	return &Cache{items: make(map[string]*list.Element), lru: list.New(), maxEntries: maxEntries, maxBytes: maxBytes, now: time.Now}
}

// Get returns a defensive response copy and refreshes its recency.
func (c *Cache) Get(key string) (Response, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	element, ok := c.items[key]
	if !ok {
		return Response{}, false
	}
	entry := element.Value.(*item)
	if !c.now().Before(entry.expiresAt) {
		c.remove(element)
		return Response{}, false
	}
	c.lru.MoveToFront(element)
	return cloneResponse(entry.response), true
}

// Set inserts a response when it fits the configured byte bound.
func (c *Cache) Set(key string, response Response, ttl time.Duration) bool {
	if ttl <= 0 {
		return false
	}
	response = cloneResponse(response)
	response.Stored = c.now()
	size := responseSize(key, response)
	if size > c.maxBytes {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if existing, ok := c.items[key]; ok {
		c.remove(existing)
	}
	entry := &item{key: key, response: response, expiresAt: c.now().Add(ttl), size: size}
	element := c.lru.PushFront(entry)
	c.items[key] = element
	c.bytes += size
	for len(c.items) > c.maxEntries || c.bytes > c.maxBytes {
		c.remove(c.lru.Back())
	}
	return true
}

// Len returns the current entry count.
func (c *Cache) Len() int { c.mu.Lock(); defer c.mu.Unlock(); return len(c.items) }

// Bytes returns the cache's accounted byte size.
func (c *Cache) Bytes() int64 { c.mu.Lock(); defer c.mu.Unlock(); return c.bytes }

func (c *Cache) remove(element *list.Element) {
	if element == nil {
		return
	}
	entry := element.Value.(*item)
	delete(c.items, entry.key)
	c.bytes -= entry.size
	c.lru.Remove(element)
}

func cloneResponse(response Response) Response {
	clone := response
	clone.Header = response.Header.Clone()
	clone.Body = append([]byte(nil), response.Body...)
	return clone
}

func responseSize(key string, response Response) int64 {
	size := len(key) + len(response.Body)
	for name, values := range response.Header {
		size += len(name)
		for _, value := range values {
			size += len(value)
		}
	}
	return int64(size)
}
