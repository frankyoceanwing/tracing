package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"time"
)

func indexHandler(w http.ResponseWriter, r *http.Request) {
	filePath := "static/index.html"
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Printf("read file[%s] failed: %s", filePath, err.Error())
		w.WriteHeader(404)
		w.Write([]byte("404 Something went wrong - " + http.StatusText(404)))
	}
	w.Write(data)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Request start...\n"))
	go func() {
		fmt.Printf("Get http://localhost:8080/async ...\n")
		http.Get("http://localhost:8080/async")
	}()
	fmt.Printf("Get http://localhost:8080/service ...\n")
	http.Get("http://localhost:8080/service")
	time.Sleep(time.Duration(rand.Intn(200)) * time.Millisecond)
	w.Write([]byte("Request done!"))
	fmt.Printf("Get http://localhost:8080/service done!\n")
}

// Mocks a service endpoint that makes a DB call
func serviceHandler(w http.ResponseWriter, r *http.Request) {
	// ...
	fmt.Printf("Get http://localhost:8080/db ...\n")
	http.Get("http://localhost:8080/db")
	time.Sleep(time.Duration(rand.Intn(200)) * time.Millisecond)
	fmt.Printf("Get http://localhost:8080/db done!\n")
	// ...
}

// Mocks a DB call
func dbHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("call DB ...\n")
	time.Sleep(time.Duration(rand.Intn(200)) * time.Millisecond)
	fmt.Printf("call DB done!\n")
	// here would be the actual call to a DB.
}

func main() {
	port := 8080
	addr := fmt.Sprintf(":%d", port)
	mux := http.NewServeMux()
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/home", homeHandler)
	mux.HandleFunc("/async", serviceHandler)
	mux.HandleFunc("/service", serviceHandler)
	mux.HandleFunc("/db", dbHandler)
	fmt.Printf("serve %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
