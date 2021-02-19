package lrucache

import (
	"fmt"
	"sync"
	"time"
)

type LRUCache interface {
	// Get returns the value associated with the specified key,
	// and promotes the entry to the head of the recenly used list.
	// Returns nil if the key is not present in the cache.
	Get(key interface{}) (value interface{})
	// Put sets the value for the specified key, and promotes the
	// entry to the head of the recently used list.
	// If Put causes the cache to exceed its capacity, the least recently
	// accessed entry is removed, i.e., the entry at the tail of the recently
	// used list.
	Put(key, value interface{})
	// Remove removes the entry for the specified key, and returns
	// the associated value, or nil if the key is not present in the cache.
	Remove(key interface{}) (value interface{})
	// Size returns the number of entries currently in the cache.
	Size() int
	// Capacity returns the maximum capacity of the cache.
	Capacity() int
	// TTLSeconds returns the TTL (time to live) value for the cache.
	// A value of 0 indicates TTL support is disabled.
	TTLSeconds() int64
	// Clear removes all entries from the cache.
	Clear()
	// Stop terminates the goroutine used to purge expired entries when TTL
	// is enabled. It has no effect if TTL is not enabled. It should be called
	// if the cache is no longer in use prior to program termination.
	Stop()
}

// New returns a new LRUCache instance with the specified capacity
// and TTL support disabled. Panics if capacity <= 0.
func New(capacity int) LRUCache {
	if capacity <= 0 {
		panic(fmt.Sprintf("invalid capacity: %d\n", capacity))
	}
	return &lrucache{
		capacity: capacity,
		hash:     make(map[interface{}]*entry),
	}
}

// NewWithTTL returns a new LRUCache instance with the specified capacity
// and ttlSeconds. Panics if capacity <= 0 or ttlSeconds < 0. A ttlSeconds
// value of 0 disables TTL support.
func NewWithTTL(capacity int, ttlSeconds int64) LRUCache {
	if capacity <= 0 {
		panic(fmt.Sprintf("invalid capacity: %d\n", capacity))
	}
	if ttlSeconds < 0 {
		panic(fmt.Sprintf("invalid ttlSeconds: %d\n", ttlSeconds))
	}
	c := &lrucache{
		capacity:   capacity,
		ttlSeconds: ttlSeconds,
		hash:       make(map[interface{}]*entry),
	}
	go pruneExpiredEntries(c)
	return c
}

func pruneExpiredEntries(c *lrucache) {
	if c.ttlSeconds == 0 || c.stopped {
		return
	}
	var stopped bool
	for {
		if stopped {
			return
		}
		time.Sleep(200 * time.Millisecond)
		func() {
			now := time.Now().Unix()
			c.Lock()
			defer c.Unlock()
			if c.stopped {
				stopped = true
				return
			}
			for e := c.head; e != nil; {
				// save next pointer, as unlink() will set to nil
				next := e.next
				if e.expireTime <= now {
					c.unlink(e)
					delete(c.hash, e.key)
				}
				e = next
			}
		}()
	}
}

type entry struct {
	next       *entry
	prev       *entry
	key        interface{}
	value      interface{}
	expireTime int64 // unix time seconds
}

type lrucache struct {
	sync.Mutex
	capacity   int
	ttlSeconds int64
	stopped    bool
	head       *entry
	tail       *entry
	hash       map[interface{}]*entry
}

func (c *lrucache) Get(key interface{}) (value interface{}) {
	c.Lock()
	defer c.Unlock()
	e := c.hash[key]
	if e == nil {
		return nil
	}
	// move entry to head of list if not already there
	if e != c.head {
		c.unlink(e)
		c.prepend(e)
	}
	return e.value
}

func (c *lrucache) Put(key, value interface{}) {
	c.Lock()
	defer c.Unlock()
	e := c.hash[key]
	if e != nil {
		e.value = value
		if c.ttlSeconds > 0 {
			e.expireTime = time.Now().Unix() + c.ttlSeconds
		}
		// move entry to head of list if not already there
		if e != c.head {
			c.unlink(e)
			c.prepend(e)
		}
		return
	}
	e = &entry{key: key, value: value}
	if c.ttlSeconds > 0 {
		e.expireTime = time.Now().Unix() + c.ttlSeconds
	}
	// insert new entry at head of list and in hash
	c.prepend(e)
	c.hash[key] = e
	// if over capacity, remove last (lru) entry
	if len(c.hash) > c.capacity {
		last := c.tail
		c.unlink(last)
		delete(c.hash, last.key)
	}
}

func (c *lrucache) Remove(key interface{}) (value interface{}) {
	c.Lock()
	defer c.Unlock()
	e := c.hash[key]
	if e == nil {
		return nil
	}
	c.unlink(e)
	delete(c.hash, key)
	return e.value
}

func (c *lrucache) Size() int {
	c.Lock()
	defer c.Unlock()
	return len(c.hash)
}

func (c *lrucache) Capacity() int {
	return c.capacity
}

func (c *lrucache) TTLSeconds() int64 {
	return c.ttlSeconds
}

func (c *lrucache) Clear() {
	c.Lock()
	defer c.Unlock()
	c.head = nil
	c.tail = nil
	c.hash = make(map[interface{}]*entry)
}

func (c *lrucache) Stop() {
	c.Lock()
	defer c.Unlock()
	c.stopped = true
}

// must be called under lock
func (c *lrucache) unlink(e *entry) {
	if e.prev == nil {
		c.head = e.next
	} else {
		e.prev.next = e.next
	}
	if e.next == nil {
		c.tail = e.prev
	} else {
		e.next.prev = e.prev
	}
	e.prev = nil
	e.next = nil
}

// must be called under lock
func (c *lrucache) prepend(e *entry) {
	e.next = c.head
	if c.head != nil {
		c.head.prev = e
	}
	c.head = e
	if c.tail == nil {
		c.tail = e
	}
}
