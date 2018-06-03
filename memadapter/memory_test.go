package memadapter

import (
	"bytes"
	"testing"
	"time"

	"github.com/JohannesKaufmann/dynamodb-cache"
)

func TestNew(t *testing.T) {
	c, err := New(time.Second*2, true)()
	if err != nil {
		t.Error(err)
	}

	k := "1"
	d := []byte("data")

	err = c.Set(k, d)
	if err != nil {
		t.Error(err)
	}
	data, err := c.Get(k)
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(data, d) {
		t.Error("got different data")
	}

	time.Sleep(time.Second * 3)

	data, err = c.Get(k)
	if err != cache.ErrNotFound {
		t.Error(err)
	}
	if data != nil {
		t.Error("got data")
	}
}

func TestGet_Found(t *testing.T) {
	c := Adapter{
		ttl: -1,
		values: map[string]*item{
			"1": {
				value:  []byte("data"),
				expire: time.Now(),
			},
		},
	}
	data, err := c.Get("1")
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(data, []byte("data")) {
		t.Fail()
	}
}
func TestGet_Expired(t *testing.T) {
	c := Adapter{
		values: map[string]*item{
			"1": {
				value:  []byte("data"),
				expire: time.Now(),
			},
		},
	}
	data, err := c.Get("1")
	if err != cache.ErrExpired {
		t.Error(err)
	}
	if data != nil {
		t.Fail()
	}
}
func TestGet_NotFound(t *testing.T) {
	c := Adapter{
		values: make(map[string]*item),
	}
	data, err := c.Get("1")
	if err != cache.ErrNotFound {
		t.Error(err)
	}
	if data != nil {
		t.Fail()
	}
}

func TestGet_DontRenew(t *testing.T) {
	old := time.Now().Add(time.Second * 2)
	c := Adapter{
		ttl:         time.Second,
		renewOnRead: false,
		values: map[string]*item{
			"1": {
				expire: old,
			},
		},
	}
	_, err := c.Get("1")
	if err != nil {
		t.Error(err)
	}
	if old != c.values["1"].expire {
		t.Error("expire did change")
	}
}
func TestGet_Renew(t *testing.T) {
	old := time.Now().Add(time.Second * 2)
	c := Adapter{
		ttl:         time.Second,
		renewOnRead: true,
		values: map[string]*item{
			"1": {
				expire: old,
			},
		},
	}
	_, err := c.Get("1")
	if err != nil {
		t.Error(err)
	}
	if old == c.values["1"].expire {
		t.Error("expire did not change")
	}
}

func TestSet(t *testing.T) {
	c := Adapter{
		values: make(map[string]*item),
	}
	err := c.Set("1", []byte("One"))
	if err != nil {
		t.Error(err)
	}
	if len(c.values) != 1 {
		t.Error("expected length of 1")
	}

	i := c.values["1"]
	if !bytes.Equal(i.value, []byte("One")) {
		t.Error("other data in map")
	}
}

func TestDel(t *testing.T) {
	c := Adapter{
		values: map[string]*item{
			"1": {
				value:  []byte("data"),
				expire: time.Now(),
			},
		},
	}
	err := c.Del("1")
	if err != nil {
		t.Error(err)
	}
	if len(c.values) != 0 {
		t.Fail()
	}
}

func TestExpire(t *testing.T) {
	now := time.Now()

	c := Adapter{
		ttl: time.Second,
		values: map[string]*item{
			"1": {
				expire: now,
			},
		},
	}
	c.deleteExpired(now.Add(-time.Second))
	if len(c.values) != 1 {
		t.Error("item deleted to early")
	}

	c.deleteExpired(now.Add(time.Second))
	if len(c.values) != 0 {
		t.Error("item deleted to soon")
	}
}
