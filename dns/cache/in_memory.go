package cache
import(
	"time"
)
type Item struct{
	Expire time.Time
	Val any
}

func (item Item) Expired() bool {
	return item.Expire.Before(time.Now())  
}

type InMemory struct {
	Data map[string]*Item
}

func NewMemoryCache() *InMemory {
	return &InMemory{
		Data: make(map[string]*Item),
	}
}

func (c *InMemory) Set(key string, value Item) {
	c.Data[key] = &value
}

func (c *InMemory) Get(key string) (any, error) {
	if item,ok:= c.Data[key];ok{
		if item.Expired(){
			_ = c.Delete(key)
			return nil,nil
		}
		return item.Val,nil
	}
	return nil,ErrKeyNotExists
}

func (c *InMemory) Delete(key string) error {
	if _, ok := c.Data[key]; ok {
		delete(c.Data, key)
		return nil
	}
	return ErrKeyNotExists
}

func (c *InMemory) Exists(key string) bool {
	if _, ok := c.Data[key]; !ok {
		return false
	}

	return true
}
