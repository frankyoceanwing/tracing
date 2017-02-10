package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"

	"github.com/frankyoceanwing/tracing/server"
	"github.com/opentracing/opentracing-go"
	"sourcegraph.com/sourcegraph/appdash"
	appdashot "sourcegraph.com/sourcegraph/appdash/opentracing"
	"sourcegraph.com/sourcegraph/appdash/traceapp"
)

func newAppdashTracer() opentracing.Tracer {
	store := appdash.NewMemoryStore()

	// Listen on any available TCP port locally.
	l, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		log.Fatal(err)
	}
	collectorPort := l.Addr().(*net.TCPAddr).Port
	collectorAddr := fmt.Sprintf(":%d", collectorPort)

	// Start an Appdash collection server that will listen for spans and
	// annotations and add them to the local collector (stored in-memory).
	cs := appdash.NewServer(l, appdash.NewLocalCollector(store))
	go cs.Start()

	// Print the URL at which the web UI will be running.
	appdashPort := 8700
	appdashURLStr := fmt.Sprintf("http://localhost:%d", appdashPort)
	appdashURL, err := url.Parse(appdashURLStr)
	if err != nil {
		log.Fatalf("Error parsing %s: %s", appdashURLStr, err)
	}
	fmt.Printf("To see your traces, go to %s/traces\n", appdashURL)

	// Start the web UI in a separate goroutine.
	tapp, err := traceapp.New(nil, appdashURL)
	if err != nil {
		log.Fatal(err)
	}
	tapp.Store = store
	tapp.Queryer = store
	go func() {
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", appdashPort), tapp))
	}()

	tracer := appdashot.NewTracer(appdash.NewRemoteCollector(collectorAddr))
	return tracer
}

func main() {
	tracer := newAppdashTracer()

	// set the tracer to be the default tracer
	opentracing.InitGlobalTracer(tracer)

	port := 8080
	addr := fmt.Sprintf(":%d", port)
	mux := http.NewServeMux()
	mux.HandleFunc("/", server.IndexHandler)
	mux.HandleFunc("/home", server.HomeHandler)
	mux.HandleFunc("/async", server.ServiceHandler)
	mux.HandleFunc("/service", server.ServiceHandler)
	mux.HandleFunc("/db", server.DBHandler)
	fmt.Printf("serve %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
