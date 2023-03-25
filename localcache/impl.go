// Package localcache provides a simple lru cache mechanism storing data in memory.
package localcache

import (
		"container/list"
		"sync"
		"time"
)

// localCache is a simple lru cache implementation with ttl.
type localCache struct {
		m     sync.Mutex
		ttl   time.Duration
		size  int
		store map[string]*list.Element
		items *list.List
}

// item is a cache item store key, value & when it's expired.
type item struct {
	  key   string
		val   interface{}
		expAt time.Time
}

// Get looks up a key's value from the cache. Returns value, true if the key was found.
// If the key is not found, or if the key is found but has expired, return nil, false.
func (c *Cache) Get(key string) (interface{}, bool) {
		c.m.Lock()
		defer c.m.Unlock()
		if ele, ok := c.store[key]; ok {
				if ele.Value.(*item).expired() {
						return nil, false
				}
				c.items.MoveToFront(ele)
				return ele.Value.(*item).val, true
		}
		return nil, false
}

// Set adds a value to the cache.  Returns true if an eviction occurred.
func (c *Cache) Set(key string, value interface{}) bool {
		c.m.Lock()
		defer c.m.Unlock()
		if ele, ok := c.store[key]; ok {
				c.items.MoveToFront(ele)
				ele.Value.(*item).val = value
				ele.Value.(*item).expAt = c.expiresAt()
				return false
		}

		new := &item{ key: key, val: value, expAt: c.expiresAt() }
		ele := c.items.PushFront(new)
		c.store[key] = ele

		if c.items.Len() > c.size {
				c.evict()
				return true
		}
		return false
}

// New returns a new localCache with the default size & ttl.
func New() (*localCache, error) {
		return &localCache{
				ttl:   30 * time.Second,
				size:  128,
				store: make(map[string]*list.Element{}),
				items: list.New()
		}
}

// evict removes the least recently used item from the cache. Not thread safe.
func (c *localCache) evict() {
		back := c.items.Back()
		if back != nil {
				c.items.Remove(back)
				delete(c.store, back.Values.(*item).key)
		}
}

func (c *localCache) expiresAt() time.Time {
		return time.Now().Add(c.ttl)
}

func (i *item) expired() bool {
		return i.expAt.After(time.Now())
}