package cache

import (
	"fmt"
	"log"
	"sync"
)

type Getter interface {
	Get(key string) ([]byte, error)
}

type GetFunc func(key string) ([]byte, error)

func (f GetFunc) Get(key string) ([]byte, error) {
	return f(key)
}

type Group struct {
	name      string
	getter    Getter
	mainCache cache
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if nil == getter {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:   name,
		getter: getter,
		mainCache: &cache{
			cacheBytes: cacheBytes,
		},
	}
	groups[name] = g

	return g
}

func GetGroup(name string) *Group {
	mu.Lock()
	g := groups[name]
	mu.RUnlock()

	return g
}

func (g *Group) get(key string) (ByteView, error) {
	if "" == key {
		return ByteView{}, fmt.Errorf("key is required")
	}

	if v, ok := g.mainCache.find(key); ok {
		log.Printf("[GeeCache] hit")
		return v, nil
	}

	return g.load(key)
}

func (g *Group) load(key string) (ByteView, error) {
	return g.getlocally(key)
}

func (g *Group) getlocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if nil != err {
		return ByteView{}, err
	}

	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}
func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.Add(key, value)
}
