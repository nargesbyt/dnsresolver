package cache

import (
	"errors"
	"fmt"
	"hash/fnv"
	//"log"
	//"os"

	//"git.mills.io/prologic/bitcask"
	//"github.com/google/uuid"
)

//var db *bitcask.Bitcask

type Cache interface {
	Set(key string, value interface{})
	Get(key string) (interface{}, error)
}

var ErrKeyNotExists = errors.New("key doesn't exist")

type InMemoryCache struct {
	Data map[string]interface{}
}

func (c *InMemoryCache) Set(key string, value interface{}) {
	c.Data[key] = value
}

func (c *InMemoryCache) Get(key string) (interface{}, error) {
	fmt.Printf("cache data ======> %#v\n", c.Data)
	if d, ok := c.Data[key]; ok {
		return d, nil
	}
	return nil, ErrKeyNotExists

}
//******************************************************************
type FileCache struct {
	Data []byte
}

func getKeyHash(key string) string {
	keyHash := fnv.New64()
	keyHash.Write([]byte(key))
	return fmt.Sprintf("%v", keyHash.Sum64())
}

/*func (fc *FileCache) Set(key string, value interface{}) {
	k := getKeyHash((key))
	if db.Has([]byte(k)) {
		return
	}

	id := uuid.New().String()
	if _, err := os.Stat(".cache/" + id); errors.Is(err, os.ErrNotExist) {
		_, err := os.Create(".cache/" + id)
		if err != nil {
			log.Println(err)
		}
		err = os.WriteFile(id, value, 0644)
		if err != nil {
			log.Println(err)
		}
	}

	db.Put([]byte(key), []byte(id))
}*/

/*func (fc *FileCache) Get(key string) (interface{}, error) {

}*/
