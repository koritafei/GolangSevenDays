package geecache

import (
	"fmt"
	"geecache/geecache/signlefight"
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
	mainCache *LruCache
	peers PeerPicker
	loader *signlefight.Group
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
		mainCache: &LruCache{
			cacheBytes: cacheBytes,
		},
		loader: &signlefight.Group{},
	}
	groups[name] = g

	return g
}


func (g *Group) RegisterPeers(peer PeerPicker){
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peer
}

func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()

	return g
}

func (g *Group) Get(key string) (ByteView, error) {
	if "" == key {
		return ByteView{}, fmt.Errorf("key is required")
	}

	if v, ok := g.mainCache.lru.Get(key); ok {
		log.Printf("[GeeCache] hit")
		return v.(ByteView), nil
	}

	return g.load(key)
}

func (g *Group) load(key string) (value ByteView, err error) {
	
	viewi, err := g.loader.Do(key,  func() (interface{}, error) {
	if g.peers != nil {
		if peer, ok := g.peers.PickPeer(key);ok {
			if value, err = g.getFromPeer(peer, key); err == nil {
				return value, nil
			}
			log.Println("[GeeCache] Failed to get from peer", err)
		}
	}
	return g.getlocally(key)
})
	if err == nil {
		return viewi.(ByteView), nil
	}
	return 
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

func (g *Group)getFromPeer(peer PeerGetter,key string) (ByteView, error) {
	bytes, err := peer.Get(g.name, key)
	if err != nil {
		return ByteView{},err
	}

	return ByteView{b:bytes}, nil

}