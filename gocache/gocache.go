package gocache

import (
	"fmt"
	"gocache/lru"
	"log"
	"sync"
)

type cache struct {
	mu sync.Mutex
	lru *lru.Cache
	cacheBytes uint64
}

func (c *cache) add(key string, val ByteView)  {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		c.lru = lru.NewCache(c.cacheBytes, nil)
	}
	c.lru.Add(key, val) // ByteView 实现了 Value 接口
}

func (c *cache) get(key string) (val ByteView, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		return
	}
	if v, ok := c.lru.Get(key); ok {
		return v.(ByteView), ok
	}
	return
}

type Getter interface {
	Get(key string) ([]byte, error)
}

type GetterFunc func(key string) ([]byte, error)

func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

type Group struct {
	name string
	getter Getter
	mainCache cache
	peers PeerPicker
}

var (
	mu  sync.RWMutex
	groups = make(map[string]*Group)
)

func NewGroup(name string, cacheBytes uint64, getter Getter) *Group  {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name: name,
		getter: getter,
		mainCache: cache{ cacheBytes: cacheBytes },
	}
	groups[name] = g
	return g
}

func GetGroup (name string) *Group {
	mu.RLock()
	defer mu.RUnlock()
	g := groups[name]
	return g
}

func (g *Group) Get(key string) (ByteView, error)  {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}
	if v, ok := g.mainCache.get(key); ok {
		log.Println("GoCache hint")
		return v, nil
	}
	return g.load(key)
}

func (g *Group) getLocally(key string) (ByteView, error)  {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

func (g *Group) populateCache(key string, val ByteView)  {
	g.mainCache.add(key, val)
}

func (g *Group) RegisterPeers(peers PeerPicker)  {
	if g.peers != nil {
		panic("Register peer call more than once")
	}
	g.peers = peers // what the shit ???
}

func (g *Group) load(key string) (val ByteView, err error) {
	if g.peers != nil {
		if peer, ok := g.peers.PickPeer(key); ok {
			if val, err := g.getFromPeer(peer, key); err == nil {
				return val, nil
			}
			log.Println("[GoCache] failed to get from peer", err)
		}
	}
	return g.getLocally(key)
}

func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error)  {
	bytes, err := peer.Get(g.name, key)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: bytes}, nil

}