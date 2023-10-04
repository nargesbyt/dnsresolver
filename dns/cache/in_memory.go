package cache

type InMemory struct {
	Data map[string]any
}

func (c *InMemory) Set(key string, value any) {
	c.Data[key] = value
}

func (c *InMemory) Get(key string) (any, error) {
	if c.Exists(key) {
		return c.Data[key], nil
	}

	return nil, ErrKeyNotExists
}

func (c *InMemory) Exists(key string) bool {
	if _, ok := c.Data[key]; !ok {
		return false
	}

	return true
}
