package cache

import (
	"encoding/json"
	"os"
	"path"
	"sync"
)

type FileCache struct {
	lock sync.RWMutex
	Dir  string
}

func NewFileCache(dir string) (*FileCache, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return nil, err
		}
	}
	f := &FileCache{
		Dir: dir,
	}
	return f, nil
}

func (f *FileCache) Exists(key string) bool {
	_, err := os.Stat(path.Join(f.Dir, key+".json"))
	return !os.IsNotExist(err)
}

func (f *FileCache) get(key string) (any, error) {

	// read cache from file
	bytes, err := os.ReadFile(path.Join(f.Dir, key+".json"))
	if err != nil {
		return nil, err
	}

	item := &Item{}
	if err = json.Unmarshal(bytes, item); err != nil {
		return nil, err
	}
	if item.Expired() {
		err = f.Delete(key)
		if err != nil {
			return nil, err
		}
		return nil,nil
	}
	return item.Val, nil

}

func (f *FileCache) Get(key string) (any, error) {
	f.lock.RLock()
	defer f.lock.RUnlock()

	v, err := f.get(key)
	return v, err

}

func (f *FileCache) set(key string, val Item) {
	// cache item data to file
	bs, err := json.Marshal(val)
	if err != nil {
		return
	}
	file := f.Dir + key + ".json"
	fileDesc, err := os.Create(file)
	if err != nil {
		return
	}
	defer fileDesc.Close()

	fileDesc.Write(bs)

}

func (f *FileCache) Set(key string, val Item) {
	f.lock.Lock()
	defer f.lock.Unlock()

	f.set(key, val)
}

func (c *FileCache) Delete(key string) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.del(key)
}

func (f *FileCache) del(key string) error {
	if f.Exists(key) {
		err := os.Remove(path.Join(f.Dir, key+".json"))
		if err != nil {
			return err
		}
		return nil
	}
	return ErrKeyNotExists
}
