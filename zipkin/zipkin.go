package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/frankyoceanwing/tracing/server"
	"github.com/opentracing/opentracing-go"
	zipkin "github.com/openzipkin/zipkin-go-opentracing"
)

func newZipkinTracer() opentracing.Tracer {
	// create a HTTP collector
	zipkinHTTPEndpoint := "http://localhost:9411/api/v1/spans"
	collector, err := zipkin.NewHTTPCollector(zipkinHTTPEndpoint)
	if err != nil {
		log.Fatalf("unable to create Zipkin HTTP collector: %+v\n", err)
	}

	// create a recorder
	debug := false
	hostPort := "127.0.0.1:0"
	serviceName := "foo"
	recorder := zipkin.NewRecorder(collector, debug, hostPort, serviceName)

	// create a tracer
	sameSpan := true
	traceID128Bit := true
	tracer, err := zipkin.NewTracer(
		recorder,
		zipkin.ClientServerSameSpan(sameSpan),
		zipkin.TraceID128Bit(traceID128Bit),
	)
	if err != nil {
		log.Fatalf("unable to create Zipkin tracer: %+v\n", err)
	}
	return tracer
}

func main() {
	tracer := newZipkinTracer()
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
