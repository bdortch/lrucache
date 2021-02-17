package lrucache

import (
	"fmt"
	"sync"
	"time"
)

type LRUCache interface {
	Get(key interface{}) (value interface{})
	Put(key, value interface{})
	Remove(key interface{}) (value interface{})
	Size() int
	Capacity() int
	TTLSeconds() int64
	Clear()
}

func New(capacity int) LRUCache {
	if capacity <= 0 {
		panic(fmt.Sprintf("invalid capacity: %d\n", capacity))
	}
	return &lrucache{
		capacity: capacity,
		hash:     make(map[interface{}]*entry),
	}
}

func NewWithTTL(capacity int, ttlSeconds int64) LRUCache {
	if capacity <= 0 {
		panic(fmt.Sprintf("invalid capacity: %d\n", capacity))
	}
	if ttlSeconds < 0 {
		panic(fmt.Sprintf("invalid ttlMillis: %d\n", ttlSeconds))
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
	for {
		time.Sleep(100 * time.Millisecond)
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
		// move entry to head of list if not already there
		if e != c.head {
			c.unlink(e)
			c.prepend(e)
		}
		return
	}
	e = &entry{key: key, value: value}
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
