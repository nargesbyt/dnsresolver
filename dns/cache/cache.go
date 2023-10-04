package cache

import (
	"errors"
)

type Cache interface {
	Set(key string, value any)
	Get(key string) (any, error)
	Exists(key string) bool
}

var ErrKeyNotExists = errors.New("key doesn't exist")
