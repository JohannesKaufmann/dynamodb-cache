package cache

import (
	"errors"

	"github.com/vmihailenco/msgpack"
)

//go:generate moq -pkg cache_test -out adapter_moq_test.go . Adapter

type Adapter interface {
	Set(key string, value []byte) error
	Get(key string) ([]byte, error)
	Del(key string) error

	// Del(keys ...string) error
	// GetMultiple, BatchGet
}

type InitAdapter func() (Adapter, error)

type Cache struct {
	adapters []Adapter
}

// New initializes a new cache with the adapters that are passed in.
func New(adapters ...InitAdapter) (*Cache, error) {
	if len(adapters) == 0 {
		return nil, errors.New("you need at least one adapter")
	}

	var c Cache
	for _, init := range adapters {
		adapter, err := init()
		if err != nil {
			return nil, err
		}

		c.adapters = append(c.adapters, adapter)
	}

	return &c, nil
}

// common errors
var (
	ErrNotFound = errors.New("item not found")
	ErrExpired  = errors.New("item found but expired")
)

/*
func GetBytes(key interface{}) ([]byte, error) {
    var buf bytes.Buffer
    enc := gob.NewEncoder(&buf)
    err := enc.Encode(key)
    if err != nil {
        return nil, err
    }
    return buf.Bytes(), nil
}
*/

// Get gets the item from the cache. It tries every adapter until
// it finds it.
func (c *Cache) Get(key string, target interface{}) error {
	var finalErr = ErrNotFound

	for _, adapter := range c.adapters {
		data, err := adapter.Get(key)

		if err != nil && (err == ErrNotFound || err == ErrExpired) {
			finalErr = err
			continue
		} else if err != nil {
			return err
		}

		return msgpack.Unmarshal(data, target)
	}

	return finalErr
}

// Set sets the value for that key in the cache.
func (c *Cache) Set(key string, value interface{}) error {
	data, err := msgpack.Marshal(value)
	if err != nil {
		return err
	}

	for _, adapter := range c.adapters {
		err := adapter.Set(key, data)
		if err != nil {
			return err
		}
	}

	return nil
}

// Del deletes the item from the cache. The item is deleted from every adapter.
func (c *Cache) Del(key string) error {
	for _, adapter := range c.adapters {
		err := adapter.Del(key)
		if err != nil {
			return err
		}
	}

	return nil
}

// - - - - - - //
/*
type Batch struct {}

func (c *Cache) Batch() *Batch {
	return &Batch{}
}
func (b *Batch) Get(target interface{}, keys ...string) error {
	return nil
}
func (b *Batch) Set(keyval ...interface{}) error {
	for i := 0; i < len(keyval); i += 2 {
		key, ok := keyval[i].(string)
		if !ok {
			return errors.New("key is not a string")
		}
		val := keyval[i+1]

		data, err := msgpack.Marshal(val)
		if err != nil {
			return err
		}
		fmt.Println(key, "->", val, data)

		var res int
		err = msgpack.Unmarshal(data, &res)
		if err != nil {
			return err
		}
		fmt.Println("res", res)
	}
	return nil
}
func (b *Batch) Del(keys ...string) error {
	return nil
}
*/
