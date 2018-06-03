package cache

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
)

// func Middleware() {
// 	// Enable line numbers in logging
// 	log.SetFlags(log.LstdFlags | log.Lshortfile)

// 	// Should say: "[date] [time] loglines.go:11: Example"
// 	log.Println("Example")
// }

func sortURLParams(URL *url.URL) {
	params := URL.Query()
	for _, param := range params {
		sort.Slice(param, func(i, j int) bool {
			return param[i] < param[j]
		})
	}
	URL.RawQuery = params.Encode()
}

type response struct {
	Code      int
	HeaderMap http.Header
	Body      []byte
}

func (r *response) fromRecorder(rec *httptest.ResponseRecorder) {
	r.Code = rec.Code
	if r.Code == 0 {
		r.Code = http.StatusOK
	}

	r.HeaderMap = rec.HeaderMap
	r.Body = rec.Body.Bytes()
}

// Cacheable contains the HTTP status codes are defined as cacheable.
// -> https://stackoverflow.com/a/39406969
var Cacheable = []int{
	http.StatusOK,
	http.StatusNonAuthoritativeInfo,
	http.StatusNoContent,
	http.StatusPartialContent,
	http.StatusMultipleChoices,
	http.StatusMovedPermanently,
	http.StatusNotFound,
	http.StatusMethodNotAllowed,
	http.StatusGone,
	http.StatusRequestURITooLong,
	http.StatusNotImplemented,
}

func (r *response) toWriter(w http.ResponseWriter, cacheHeader string) {
	for key := range r.HeaderMap {
		w.Header().Set(key, r.HeaderMap.Get(key))
	}
	if cacheHeader != "" {
		w.Header().Set("X-Cache", cacheHeader)
	}
	w.Header().Set("X-Cache", cacheHeader)
	w.WriteHeader(r.Code)
	w.Write(r.Body)
}

func (c *Cache) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				next.ServeHTTP(w, r)
				return
			}

			sortURLParams(r.URL)
			key := r.URL.String()

			var resp response
			err := c.Get(key, &resp)

			var cacheHeader string
			if err == nil {
				resp.toWriter(w, "HIT")
				return
			} else if err == ErrNotFound {
				cacheHeader = "MISS"
			} else if err == ErrExpired {
				cacheHeader = "EXPIRED"
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			rec := httptest.NewRecorder()
			next.ServeHTTP(rec, r)

			resp.fromRecorder(rec)

			var cacheable = false
			for _, code := range Cacheable {
				if code == resp.Code {
					cacheable = true
				}
			}
			if cacheable {
				go func() {
					err = c.Set(key, resp)
					if err != nil {
						fmt.Println("set err:", err)
					}
				}()
			} else {
				cacheHeader = ""
			}

			resp.toWriter(w, cacheHeader)
			return

		})
	}
}
