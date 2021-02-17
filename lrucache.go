package lrucache

import (
	"sync"
)

type LRUCache interface {
	Get(key interface{}) (value interface{})
	Put(key, value interface{})
	Remove(key interface{}) (value interface{})
	Contains(key interface{}) bool
	Size() int
	Capacity() int
	TTLMillis() int
	Clear()
}

func New(capacity int) LRUCache {
	// return &lrucache{
	// 	capacity: capacity,
	// }
	panic("unimplemented")
}

func NewWithTTL(capacity, ttlMillis int) LRUCache {
	panic("unimplemented")
}

type entry struct {
	next          *entry
	prev          *entry
	key           interface{}
	value         interface{}
	timeoutMillis int
}

type lrucache struct {
	sync.Mutex
	capacity  int
	ttlMillis int
	head      *entry
	tail      *entry
	hash      map[interface{}]*entry
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

func (c *lrucache) Size() int {
	c.Lock()
	defer c.Unlock()
	return len(c.hash)
}

func (c *lrucache) Capacity() int {
	return c.capacity
}

func (c *lrucache) TTLMillis() int {
	return c.ttlMillis
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
