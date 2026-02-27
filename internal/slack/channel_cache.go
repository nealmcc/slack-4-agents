package slack

import "sync"

type channelCache struct {
	mu    sync.RWMutex
	items map[string]string
}

func newChannelCache() *channelCache {
	return &channelCache{
		items: make(map[string]string),
	}
}

func (c *channelCache) Get(name string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	id, ok := c.items[name]
	return id, ok
}

func (c *channelCache) Set(name, id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[name] = id
}
