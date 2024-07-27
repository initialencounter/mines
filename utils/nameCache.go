package utils

import "sync"

type DualMap struct {
	idToName map[int]string
	nameToId map[string]int
}

type NameCache struct {
	mu    sync.RWMutex
	store DualMap
}

func NewNameCache() *NameCache {
	return &NameCache{
		store: DualMap{
			idToName: make(map[int]string),
			nameToId: make(map[string]int),
		},
	}
}

func (c *NameCache) Set(id int, name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store.nameToId[name] = id
	c.store.idToName[id] = name
}

func (c *NameCache) GetName(id int) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	name, ok := c.store.idToName[id]
	return name, ok
}

func (c *NameCache) GetId(name string) (int, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	id, ok := c.store.nameToId[name]
	return id, ok
}

func (c *NameCache) Delete(id int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var name = c.store.idToName[id]
	delete(c.store.idToName, id)
	delete(c.store.nameToId, name)
}
