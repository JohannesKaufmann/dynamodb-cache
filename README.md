# dynamodb-cache
Cache with **Memory** and **DynamoDB** Adapters.

![gopher with a hat that has wires going to a dynamodb database on the floor](/logo.png)


## Installation
```
go get github.com/JohannesKaufmann/dynamodb-cache
```

## Usage

Create a DynamoDB table (for example called `Cache`) and pass the name
to `dynadapter.New` as the second element.
- The hash key needs to be named `Key`
- [Enable Time To Live](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/time-to-live-ttl-how-to.html) for the attribute `TTL`

```go
type person struct {
  Name string
  Age  int
}
// - - - initialize - - - //
c, err := cache.New(
  memadapter.New(time.Hour, false),
  dynadapter.New(db, "Cache", time.Hour*24*7),
  // the order matters: items are saved (Set) and deleted (Del)
  // from to both but retrieved (Get) from the first adapter
  // (memadapter) first. If not found it tries the next
  // adapter (dynadapter).
)
if err != nil {
  log.Fatal(err)
}

// - - - set - - - //
john := person{
  Name: "John",
  Age:  19,
}
// set can be called with strings, ints, structs, maps, slices, ...
err = c.Set("1234", john)
if err != nil {
  log.Fatal(err)
}
fmt.Println("set person")

// - - - get - - - //
var p person
// remember to pass in a pointer
err = c.Get("1234", &p)
if err != nil {
  log.Fatal(err)
}
fmt.Printf("get person: %+v \n", p)

// - - - del - - - //
err = c.Del("123")
if err != nil {
  log.Fatal(err)
}
```

`Get` returns an error that is one of the following:
- `cache.ErrNotFound` if the item was not found in ANY of the adapters.
- `cache.ErrExpired` if the item was found but already expired (expired but not yet deleted). Remember that for DynamoDB it can take up to [48h](https://stackoverflow.com/a/45204322) for the deletion to happen.
- other error (typically network error)


## Middleware

```go
r := chi.NewRouter()
r.Use(middleware.Logger)
r.Use(middleware.Recoverer)
r.Use(c.Middleware()) // cache middleware

r.Get("/", func(w http.ResponseWriter, r *http.Request) {
  w.Write([]byte("welcome"))
})
r.Get("/hi", func(w http.ResponseWriter, r *http.Request) {
  var name = "World"

  w.Write([]byte("Hello " + name))
})
http.ListenAndServe(":3000", r)
```

## Related Projects

- [victorspringer/http-cache](https://github.com/victorspringer/http-cache) High performance Golang HTTP middleware for server-side application layer caching, ideal for REST APIs. ([feedback on reddit](https://www.reddit.com/r/golang/comments/8dlhbg/http_caching_middleware_feedbacks_please/))
