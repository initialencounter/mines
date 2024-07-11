package utils

import "sync"

type NameCache struct {
	mu    sync.RWMutex
	store map[int]string
}

func NewNameCache() *NameCache {
	return &NameCache{
		store: make(map[int]string),
	}
}

func (c *NameCache) Set(id int, name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[id] = name
}

func (c *NameCache) Get(id int) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	name, ok := c.store[id]
	return name, ok
}

func (c *NameCache) Delete(id int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.store, id)
}
