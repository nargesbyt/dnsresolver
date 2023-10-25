package cache

import (
	"errors"
)

type Cache interface {
	Set(key string, value Item)
	Get(key string) (any, error)
	Exists(key string) bool
	Delete(key string) error
}

var ErrKeyNotExists = errors.New("key doesn't exist")
