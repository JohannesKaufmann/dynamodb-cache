package cache_test

// TODO: // +build unit
import (
	"errors"
	"strings"
	"testing"

	"github.com/vmihailenco/msgpack"

	cache "github.com/JohannesKaufmann/dynamodb-cache"
)

func TestNew_NoAdapters(t *testing.T) {
	_, err := cache.New()
	if err == nil {
		t.Error("expected New to fail because there is not adapter")
	}
}
func TestNew_OneAdapterWithError(t *testing.T) {
	e := errors.New("some error")
	adapter := func() cache.InitAdapter {
		return func() (cache.Adapter, error) {
			return nil, e
		}
	}
	_, err := cache.New(adapter())
	if err != e {
		t.Error("expected New to fail because the adapter returned an error")
	}
}

func TestGet_Fallthrough(t *testing.T) {
	var mock1 *AdapterMock
	var mock2 *AdapterMock

	mock1 = &AdapterMock{
		GetFunc: func(key string) ([]byte, error) {
			if len(mock2.GetCalls()) != 0 {
				t.Error("expected mock1 to be called before mock2")
			}
			return nil, cache.ErrNotFound
		},
	}
	mock2 = &AdapterMock{
		GetFunc: func(key string) ([]byte, error) {
			data, err := msgpack.Marshal("One")
			if err != nil {
				t.Error(err)
			}
			return data, nil
		},
	}

	adapter1 := func() (cache.Adapter, error) {
		return mock1, nil
	}
	adapter2 := func() (cache.Adapter, error) {
		return mock2, nil
	}
	c, err := cache.New(adapter1, adapter2)
	if err != nil {
		t.Error(err)
	}

	var target string
	err = c.Get("1", &target)
	if err != nil {
		t.Error(err)
	}
	if target != "One" {
		t.Errorf("expected 'One' but got '%s'", target)
	}
}

func TestGet_BreakOnError(t *testing.T) {
	var e = errors.New("some error")
	mock1 := &AdapterMock{
		GetFunc: func(key string) ([]byte, error) {
			return nil, e
		},
	}
	adapter1 := func() (cache.Adapter, error) {
		return mock1, nil
	}
	adapter2 := func() (cache.Adapter, error) {
		return nil, nil
	}
	c, err := cache.New(adapter1, adapter2)
	if err != nil {
		t.Error(err)
	}

	var target string
	err = c.Get("1", &target)
	if err != e {
		t.Error(err)
	}
}

func TestGet_FinalError(t *testing.T) {
	var mock1 *AdapterMock
	var mock2 *AdapterMock

	mock1 = &AdapterMock{
		GetFunc: func(key string) ([]byte, error) {
			return nil, cache.ErrNotFound
		},
	}
	mock2 = &AdapterMock{
		GetFunc: func(key string) ([]byte, error) {
			return nil, cache.ErrExpired
		},
	}

	adapter1 := func() (cache.Adapter, error) {
		return mock1, nil
	}
	adapter2 := func() (cache.Adapter, error) {
		return mock2, nil
	}
	c, err := cache.New(adapter1, adapter2)
	if err != nil {
		t.Error(err)
	}

	var target string
	err = c.Get("1", &target)
	if err != cache.ErrExpired {
		t.Error(err)
	}
}

func TestSet_SetEverywhere(t *testing.T) {

	mock1 := &AdapterMock{
		SetFunc: func(key string, data []byte) error {
			if key != "1" || data == nil {
				t.Fail()
			}
			return nil
		},
	}
	mock2 := &AdapterMock{
		SetFunc: func(key string, data []byte) error {
			if key != "1" || data == nil {
				t.Fail()
			}
			return nil
		},
	}

	adapter1 := func() (cache.Adapter, error) {
		return mock1, nil
	}
	adapter2 := func() (cache.Adapter, error) {
		return mock2, nil
	}
	c, err := cache.New(adapter1, adapter2)
	if err != nil {
		t.Error(err)
	}

	err = c.Set("1", "One")
	if err != nil {
		t.Error(err)
	}

	if len(mock1.SetCalls()) != 1 {
		t.Error("expected set1 to be called once")
	}
	if len(mock2.SetCalls()) != 1 {
		t.Error("expected set2 to be called once")
	}
}

func TestSet_SetError(t *testing.T) {
	var e = errors.New("some error")
	mock1 := &AdapterMock{
		SetFunc: func(key string, data []byte) error {
			return e
		},
	}
	adapter1 := func() (cache.Adapter, error) {
		return mock1, nil
	}
	c, err := cache.New(adapter1)
	if err != nil {
		t.Error(err)
	}

	err = c.Set("1", "One")
	if err != e {
		t.Error(err)
	}
}
func TestSet_MarshalError(t *testing.T) {
	adapter1 := func() (cache.Adapter, error) {
		return nil, nil
	}
	c, err := cache.New(adapter1)
	if err != nil {
		t.Error(err)
	}

	err = c.Set("1", make(chan int))
	if !strings.Contains(err.Error(), "msgpack") {
		t.Error(err)
	}
}

func TestDel_DelEverywhere(t *testing.T) {

	mock1 := &AdapterMock{
		DelFunc: func(key string) error {
			if key != "1" {
				t.Fail()
			}
			return nil
		},
	}
	mock2 := &AdapterMock{
		DelFunc: func(key string) error {
			if key != "1" {
				t.Fail()
			}
			return nil
		},
	}

	adapter1 := func() (cache.Adapter, error) {
		return mock1, nil
	}
	adapter2 := func() (cache.Adapter, error) {
		return mock2, nil
	}
	c, err := cache.New(adapter1, adapter2)
	if err != nil {
		t.Error(err)
	}

	err = c.Del("1")
	if err != nil {
		t.Error(err)
	}

	if len(mock1.DelCalls()) != 1 {
		t.Error("expected del1 to be called once")
	}
	if len(mock2.DelCalls()) != 1 {
		t.Error("expected del2 to be called once")
	}
}

func TestDel_DelError(t *testing.T) {
	var e = errors.New("some error")
	mock1 := &AdapterMock{
		DelFunc: func(key string) error {
			return e
		},
	}
	adapter1 := func() (cache.Adapter, error) {
		return mock1, nil
	}
	c, err := cache.New(adapter1)
	if err != nil {
		t.Error(err)
	}

	err = c.Del("1")
	if err != e {
		t.Error(err)
	}
}
