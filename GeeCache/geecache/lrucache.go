package geecache

import (
	"container/list"
	"sync"
)

type LruCache struct {
	lru        *Cache
	mutex      sync.Mutex
	cacheBytes int64
}

type Cache struct {
	maxBytes  int64
	nbytes    int64
	ll        *list.List
	cache     map[string]*list.Element
	OnEvicted func(key string, value Value)
}

type entry struct {
	key   string
	value Value
}

type Value interface {
	Len() int
}

func (c *LruCache) Add(key string, value ByteView) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if nil == c.lru {
		c.lru = New(c.cacheBytes, nil)
	}

	c.lru.Add(key, value)
}

func (c *LruCache) Get(key string) (value Value, ok bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if nil == c.lru {
		return
	}

	if v, ok := c.lru.Get(key); ok {
		return v.(ByteView), ok
	}

	return
}

// New

func New(maxBytes int64, OnEvicted func(key string, value Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		OnEvicted: OnEvicted,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
	}
}

// find
func (c *Cache) Get(key string) (value Value, ok bool) {
	if elem, ok := c.cache[key]; ok {
		c.ll.MoveToFront(elem)
		kv := elem.Value.(*entry)

		return kv.value, ok
	}

	return
}

// Remove
func (c *Cache) RemoveOldest() {
	elem := c.ll.Back()
	if elem != nil {
		c.ll.Remove(elem)
		kv := elem.Value.(*entry)
		delete(c.cache, kv.key)
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

func (c *Cache) Add(key string, value Value) {
	if elem, ok := c.cache[key]; ok {
		// exists
		c.ll.MoveToFront(elem)
		kv := elem.Value.(*entry)
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		elem := c.ll.PushFront(&entry{key: key, value: value})
		c.cache[key] = elem
		c.nbytes += int64(value.Len()) + int64(len(key))
	}

	if c.maxBytes != 0 && c.nbytes > c.maxBytes {
		c.RemoveOldest()
	}

}

func (c *Cache) Len() int {
	return c.ll.Len()
}
