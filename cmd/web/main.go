package main

import (
	"flag"
	"log"
	"net/http"
)

func main() {
	addr := flag.String("addr", ":4000", "HTTP network address")

	flag.Parse()

	mux := http.NewServeMux()

	fileServer := http.FileServer(http.Dir("./ui/static/"))

	mux.Handle("GET /static/", http.StripPrefix("/static", fileServer))

	mux.HandleFunc("GET /{$}", home)
	mux.HandleFunc("GET /snippet/view/{id}", snippetView)
	mux.HandleFunc("GET /snippet/create", snippetCreate)
	mux.HandleFunc("POST /snippet/create", snippetCreatePost)

	log.Printf("Starting server on :%s\n", *addr)

	err := http.ListenAndServe(*addr, mux)
	log.Fatal(err)
}
