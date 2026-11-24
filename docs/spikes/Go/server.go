package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func hello(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello %s", r.URL.Path)
}

func ping(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "{\"status\":\"ok\"}")
}

func main() {
	port := os.Getenv("PORT")
	http.HandleFunc("/hello", hello)
	http.HandleFunc("/ping", ping)
	log.Fatal(http.ListenAndServe(":" + port, nil))
}
