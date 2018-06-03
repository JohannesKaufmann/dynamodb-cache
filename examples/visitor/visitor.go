package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	cache "github.com/JohannesKaufmann/dynamodb-cache"
	"github.com/JohannesKaufmann/dynamodb-cache/memadapter"
	randomdata "github.com/Pallinder/go-randomdata"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/tjarratt/babble"
)

var articles = make(map[string]string)

func init() {
	babbler := babble.NewBabbler()
	for i := 0; i < 10; i++ {
		id := babbler.Babble()
		text := randomdata.Paragraph()

		articles[id] = text
	}
}

type visitor struct {
	Articles map[string]struct{}
}

type contextKey string

var (
	contextKeyReadArticles = contextKey("read-articles")
)

func setReadArticles(ctx context.Context, req *http.Request, num int) context.Context {
	return context.WithValue(ctx, contextKeyReadArticles, num)
}
func getReadArticles(ctx context.Context) (int, bool) {
	num, ok := ctx.Value(contextKeyReadArticles).(int)
	return num, ok
}

func trackVisitors(c *cache.Cache) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr

			var v visitor
			err := c.Get(ip, &v)
			if err != nil && (err == cache.ErrNotFound || err == cache.ErrExpired) {
				v.Articles = make(map[string]struct{})
			} else if err != nil {
				log.Fatal(err)
			}

			v.Articles[r.URL.Path] = struct{}{}
			ctx := setReadArticles(r.Context(), r, len(v.Articles))

			fmt.Printf("%+v\n", v)
			go func() {
				err = c.Set(ip, v)
				if err != nil {
					log.Fatal(err)
				}
			}()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
func articleHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	count, ok := getReadArticles(r.Context())
	if !ok {
		fmt.Println("not ok")
	}

	if count > 5 {
		w.Write([]byte("You have reached the limit."))
		return
	}

	data := map[string]interface{}{
		"ID": id,
		"Info": fmt.Sprintf(
			`You have %d of %d free articles remaining.`, 5-count, 5,
		),
		"Text": articles[id],
	}
	html := `
	<html>
		<body>
			<h1>{{ .ID }}</h1>
			<h3>{{ .Info }}</h3>

			<p>{{ .Text }}</p>
		</body>
	</html>
	`
	t := template.New("")
	t = template.Must(t.Parse(html))
	err := t.Execute(w, data)
	if err != nil {
		log.Fatal(err)
	}
}

func articlesHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Articles": articles,
	}
	html := `
	<html>
		<body>
			<h1>Articles</h1>
			<ul>
				{{ range $key, $value := .Articles }}
					<li>
						<a href="/article/{{ $key }}">{{ $key }}</a>
					</li>
				{{ end }}
			</ul>
		</body>
	</html>
	`
	t := template.New("")
	t = template.Must(t.Parse(html))
	err := t.Execute(w, data)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	c, err := cache.New(
		memadapter.New(time.Minute*10, false),
	)
	if err != nil {
		log.Fatal(err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", articlesHandler)

	track := trackVisitors(c)
	r.Handle("/article/{id}", track(http.HandlerFunc(articleHandler)))

	http.ListenAndServe(":3000", r)
}
