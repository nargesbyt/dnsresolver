package cache

import (
	"testing"
	"time"
)

func TestBasic(t *testing.T) {
	dns := NewMemoryCache()
	value := Item{
		Val:    "17.122.168.10",
		Expire: time.Now().Add(2 * time.Hour),
	}
	dns.Set("com", value)
	ip, err := dns.Get("com")
	if err != nil {
		t.Error("com was not found")
	}
	if ip == nil {
		t.Error("dns[.com] is nil")
	}
	if ip != "17.122.168.10" {
		t.Error("dns[.com]!= 17.122.168.10")
	}
}
func TestGet(t *testing.T) {
	fileCache, _ := NewFileCache("/tmp")
	getTests := []struct {
		name  string
		cache Cache
	}{
		{name: "in_memory", cache: NewMemoryCache()},
		{name: "file_cache", cache: fileCache},
	}

	for _, tt := range getTests {
		t.Run(tt.name, func(t *testing.T) {
			tt.cache.Set("com", Item{Val: "192.158.0.11", Expire: time.Now().Add(time.Hour * 2)})
			got, err := tt.cache.Get("com")
			if err == ErrKeyNotExists && got != "192.158.0.11" {
				t.Errorf("%#v got %s want \"192.158.0.11\"", tt.cache, got)
			}
		})
	}
}
func TestExists(t *testing.T) {
	item := Item{Val: "128.16.0.14", Expire: time.Now().Add(time.Minute * 30)}

	fileCache, _ := NewFileCache("tmp/")
	fileCache.Set("org", item)

	memCache := NewMemoryCache()
	memCache.Set("org", item)

	existTests := []struct {
		name  string
		cache Cache
		key   string
		exist bool
	}{
		{name: "key ir doesn't exist", cache: memCache, key: "ir", exist: false},
		{name: "key ir doesn't exist in disk", cache: fileCache, key: "ir", exist: false},
		{name: "key org exist in memory", cache: memCache, key: "org", exist: true},
		{name: "key org exist in disk", cache: fileCache, key: "org", exist: true},
	}
	for _, tt := range existTests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.exist != tt.cache.Exists(tt.key) {
				t.Errorf("%#v existence of key %s is %t but is evaluated %t ", tt.cache, tt.key, tt.exist, tt.cache.Exists(tt.key))
			}

		})
	}

}
func TestDelete(t *testing.T) {
	item := Item{Val: "128.16.0.14", Expire: time.Now().Add(time.Second * 10)}

	fileCache, _ := NewFileCache("tmp/")
	fileCache.Set("nl", item)

	memCache := NewMemoryCache()
	memCache.Set("nl", item)

	deleteTests := []struct {
		name  string
		cache Cache
		key   string
	}{
		{name: "key has deleted", cache: memCache, key: "nl"},
		{name: "key has deleted from disk", cache: fileCache, key: "nl"},
	}
	var d time.Duration = 1000
	for _, tt := range deleteTests {
		t.Run(tt.name, func(t *testing.T) {
			time.Sleep(d)
			val, err := tt.cache.Get(tt.key)
			if val != nil && err != nil {
				t.Errorf("%#v key %s should be deleted but it exist with value %#v", tt.cache, tt.key, val)
			}

		})

	}

}
