package main

import (
	"log"
	"net/http"
	"time"

	"github.com/JohannesKaufmann/dynamodb-cache"
	"github.com/JohannesKaufmann/dynamodb-cache/memory"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

func main() {
	/*
		strconv = str conv = string conversions

		memory adapter
		dynamodb adapter

		mem
		dyn

		dyn_adapt
		dyn_adpt
		dyn_adptr
		dyn_ad

		memadapt
		dynadapt

		memoryadapter
		dynamoadapter


		memadptr
		dyndptr

		-> memadapter
		-> dynadapter
	*/
	c, err := cache.New(
		memadapter.New(time.Minute*10, true),
	)
	if err != nil {
		log.Fatal(err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(c.Middleware())
	r.Use(middleware.Recoverer)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("welcome"))
	})
	r.Get("/hi", func(w http.ResponseWriter, r *http.Request) {
		var name = "World"

		w.Write([]byte("Hello " + name))
	})
	http.ListenAndServe(":3000", r)
}
