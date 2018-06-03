package cache_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	cache "github.com/JohannesKaufmann/dynamodb-cache"
	"github.com/JohannesKaufmann/dynamodb-cache/memory"
)

func TestMiddleware(t *testing.T) {
	ttl := time.Second / 2
	c, err := cache.New(
		memadapter.New(ttl, false),
	)
	if err != nil {
		t.Error(err)
	}
	m := c.Middleware()

	var i int
	var values = []string{"first call", "second call", "third call"}
	fn := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Key", "Value")
		w.Write([]byte(values[i]))

		i++
	}
	handler := http.HandlerFunc(fn)

	ts := httptest.NewServer(m(handler))
	defer ts.Close()

	get := func(expectedCacheHeader string, expectedBody string) {
		res, err := http.Get(ts.URL)
		if err != nil {
			t.Error(err)
			return
		}

		// for key, val := range res.Header {
		// 	fmt.Println("\t", key, "->", val)
		// }

		if res.Header.Get("Key") != "Value" {
			t.Error("header 'Key' is different or missing")
		}
		if val := res.Header.Get("X-Cache"); val != "HIT" && val != "MISS" && val != "EXPIRED" {
			t.Errorf("unexpected 'X-Cache' header: %s", val)
		}
		if val := res.Header.Get("X-Cache"); val != expectedCacheHeader {
			t.Errorf("expected %s but got %s", expectedCacheHeader, val)
		}

		defer res.Body.Close()
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			t.Error(err)
		}
		if string(body) != expectedBody {
			t.Error("got different body than expected")
		}
	}
	get("MISS", values[0])
	get("HIT", values[0])
	get("HIT", values[0])
	time.Sleep(ttl)
	get("EXPIRED", values[1])
	get("HIT", values[1])
	time.Sleep(memadapter.CleanupInterval)
	get("MISS", values[2])
	get("HIT", values[2])
}

func TestMiddleware_Post(t *testing.T) {
	c, err := cache.New(
		memadapter.New(time.Second*2, false),
	)
	if err != nil {
		t.Error(err)
	}
	m := c.Middleware()

	var i int
	var values = []string{"first call", "second call", "third call"}
	fn := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Key", "Value")
		w.Write([]byte(values[i]))

		i++
	}
	handler := http.HandlerFunc(fn)

	ts := httptest.NewServer(m(handler))
	defer ts.Close()

	get := func(expectedBody string) {
		res, err := http.Post(ts.URL, "", nil)
		if err != nil {
			t.Error(err)
			return
		}

		if res.Header.Get("Key") != "Value" {
			t.Error("header 'Key' is different or missing")
		}
		if res.Header.Get("X-Cache") != "" {
			t.Error("X-Cache is not empty")
		}

		defer res.Body.Close()
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			t.Error(err)
		}
		if string(body) != expectedBody {
			t.Error("got different body than expected")
		}
	}
	get(values[0])
	get(values[1])
	get(values[2])
}

func TestMiddleware_QuerySort(t *testing.T) {
	c, err := cache.New(
		memadapter.New(time.Second*2, false),
	)
	if err != nil {
		t.Error(err)
	}
	m := c.Middleware()

	var i int
	var values = []string{"first call"}
	fn := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Key", "Value")
		w.Write([]byte(values[i]))

		i++
	}
	handler := http.HandlerFunc(fn)

	ts := httptest.NewServer(m(handler))
	defer ts.Close()

	get := func(query string, expectedCacheHeader string) {
		res, err := http.Get(ts.URL + "?" + query)
		if err != nil {
			t.Error(err)
			return
		}

		if res.Header.Get("Key") != "Value" {
			t.Error("header 'Key' is different or missing")
		}
		if res.Header.Get("X-Cache") != expectedCacheHeader {
			t.Error("X-Cache is different")
		}

		defer res.Body.Close()
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			t.Error(err)
		}
		if string(body) != values[0] {
			t.Error("got different body than expected")
		}
	}
	get("a=0&b=1&c=2", "MISS")
	get("b=1&a=0&c=2", "HIT")
	get("c=2&a=0&b=1", "HIT")
}

func TestMiddleware_StatusCode(t *testing.T) {
	// TODO: for example TemporaryRedirect should not be cached
	c, err := cache.New(
		memadapter.New(time.Second*2, false),
	)
	if err != nil {
		t.Error(err)
	}
	m := c.Middleware()

	fn := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Key", "Value")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("text"))
	}
	handler := http.HandlerFunc(fn)

	ts := httptest.NewServer(m(handler))
	defer ts.Close()

	get := func(expectedCacheHeader string) {
		res, err := http.Get(ts.URL)
		if err != nil {
			t.Error(err)
			return
		}
		if res.StatusCode != http.StatusBadRequest {
			t.Error("wrong status code")
		}

		if res.Header.Get("Key") != "Value" {
			t.Error("header 'Key' is different or missing")
		}
		if res.Header.Get("X-Cache") != expectedCacheHeader {
			t.Error("X-Cache is different")
		}

		defer res.Body.Close()
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			t.Error(err)
		}
		if string(body) != "text" {
			t.Error("got different body than expected")
		}
	}
	get("")
	get("")
	get("")
}
