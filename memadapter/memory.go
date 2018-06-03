package memadapter

import (
	"fmt"
	"sync"
	"time"

	"github.com/JohannesKaufmann/dynamodb-cache"
)

type item struct {
	value  []byte
	expire time.Time
}

func (i item) isExpired(now time.Time) bool {
	return now.After(i.expire)
}

type Adapter struct {
	values map[string]*item
	m      sync.RWMutex

	ttl         time.Duration
	renewOnRead bool
}

// -> https://stackoverflow.com/a/25487392

const NoExpiration time.Duration = -1
const CleanupInterval = time.Second * 2

func New(ttl time.Duration, renewOnRead bool) cache.InitAdapter {
	return func() (cache.Adapter, error) {
		i := &Adapter{
			values:      make(map[string]*item),
			ttl:         ttl,
			renewOnRead: renewOnRead,
		}

		if ttl != NoExpiration {
			go func() {
				ticker := time.NewTicker(CleanupInterval)

				for {
					select {
					case time := <-ticker.C:
						i.deleteExpired(time)
						// case <-i.stop:
						// 	ticker.Stop()
						// 	return
					}
				}
			}()
		}

		return i, nil
	}
}

// func NewWithRenew(ttl time.Duration)cache.InitAdapter {
// 	return func() (cache.Adapter, error) {
// 		return nil,nil
// 	}
// }

func (a Adapter) deleteExpired(now time.Time) {
	a.m.Lock()
	for key, v := range a.values {
		if v.isExpired(now) {
			// if now.After(v.expire) {
			fmt.Println("cleanup: DELETE", key)
			delete(a.values, key)
		} else {
			// fmt.Println("cleanup: DONT DELETE")
		}
	}
	a.m.Unlock()
}

func (a Adapter) Get(key string) ([]byte, error) {
	a.m.RLock()
	defer a.m.RUnlock()

	if it, ok := a.values[key]; ok {
		if a.ttl != NoExpiration && it.isExpired(time.Now()) {
			return nil, cache.ErrExpired
		}
		if a.renewOnRead {
			it.expire = time.Now().Add(a.ttl)
		}

		return it.value, nil
	}

	return nil, cache.ErrNotFound
}

func (a Adapter) Set(key string, data []byte) error {
	a.m.Lock()
	defer a.m.Unlock()

	a.values[key] = &item{
		value:  data,
		expire: time.Now().Add(a.ttl),
	}

	return nil
}
func (a Adapter) Del(key string) error {
	a.m.Lock()
	defer a.m.Unlock()

	delete(a.values, key)

	return nil
}
