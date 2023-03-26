// Package localcache provides a simple lru cache mechanism storing data in memory.
package localcache

import (
		"container/list"
		"container/heap"
		"sync"
		"time"
)

// Cache is an interface for cache implementation.
type Cache interface {
		Get(key string) (interface{}, bool)
		Set(key string, value interface{})
}

// localCache is a simple lru cache implementation with ttl.
type localCache struct {
		m        sync.Mutex
		ttl      time.Duration
		cap      int
		size     int
		store    map[string]*list.Element
		lrulist  *list.List
		exppq		 priorityQueue
}

// item is a cache item store key, value, when it's expired and index for priority queue.
type cacheItem struct {
	  key   string
		val   interface{}
		exp   int64
		idx	  int
}

// Get looks up a key's value from the cache. Returns value, true if the key was found.
// If the key is not found, or if the key is found but has expired, return nil, false.
func (c *localCache) Get(key string) (interface{}, bool) {
		c.m.Lock()
		defer c.m.Unlock()

		if ele, ok := c.store[key]; ok {
				if ele.Value.(*cacheItem).expired() {
						return nil, false
				}
				c.lrulist.MoveToFront(ele)
				return ele.Value.(*cacheItem).val, true
		}
		return nil, false
}

// Set adds a value to the cache. If cache is full, it'll try to delete expired items first
// and then delete the least recently used item.
func (c *localCache) Set(key string, value interface{}) {
		c.m.Lock()
		defer c.m.Unlock()

		if c.size == c.cap {
				c.evictExpired()
		}
		if c.size == c.cap {
				c.evictLRU()
		}

		exp := time.Now().Add(c.ttl).UnixNano()
		if ele, ok := c.store[key]; ok {
				c.lrulist.MoveToFront(ele)
				c.exppq.reset(ele.Value.(*cacheItem), value, exp)
				return
		}

		item := &cacheItem{
				key: key,
				val: value,
				exp: exp,
		}
		ele := c.lrulist.PushFront(item)
		heap.Push(&c.exppq, item)
		c.store[key] = ele
		c.size++
}

// New returns a new localCache with the default capacity & ttl.
func New() *localCache {
		return &localCache{
				ttl:     30 * time.Second,
				cap:     128,
				store:   make(map[string]*list.Element, 128),
				lrulist: list.New(),
				exppq:   make(priorityQueue, 0),
		}
}

// evictExpired removes all expired items from the cache. Not thread safe.
func (c *localCache) evictExpired() {
		if c.exppq.Len() > 0 && c.exppq[0].expired() {
				heap.Pop(&c.exppq)
		}
}

// evictLRU removes the least recently used item from the cache. Not thread safe.
func (c *localCache) evictLRU() {
		back := c.lrulist.Back()
		if back != nil {
				c.lrulist.Remove(back)
				delete(c.store, back.Value.(*cacheItem).key)
		}
}

// expired returns true if the item has expired.
func (item *cacheItem) expired() bool {
		return time.Now().UnixNano() > item.exp
}
