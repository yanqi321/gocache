package lru

import "container/list"

type Cache struct {
	maxBytes uint64 // 最大内存
	nBytes uint64 // 当前内存
	ll *list.List // 键值对列表
	cache map[string]*list.Element
	OnEvicted func(key string, value Value)
}

type entry struct {
	key string
	value Value
}

type Value interface {
	Len() int
}

func NewCache(maxBytes uint64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes: maxBytes,
		ll: list.New(),
		cache: make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		return kv.value, true
	}
	return
}

func (c *Cache) ReomveOldest()  {
	ele := c.ll.Back()
	if ele != nil {
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		delete(c.cache, kv.key)
		c.nBytes -= uint64(len(kv.key)) + uint64(kv.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key,kv.value)
		}
	}
}

func (c *Cache) Add(key string, val Value)  {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		c.nBytes += uint64(val.Len()) - uint64(kv.value.Len())
		kv.value = val
	} else {
		ele := c.ll.PushFront(&entry{key, val})
		c.cache[key] = ele
		c.nBytes += uint64(len(key)) + uint64(val.Len())
	}
	for c.maxBytes != 0 && c.maxBytes < c.nBytes {
		c.ReomveOldest()
	}
}

func (c *Cache) Len() int {
	return c.ll.Len()
}