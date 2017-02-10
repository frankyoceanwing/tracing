package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	zipkin "github.com/openzipkin/zipkin-go-opentracing"
)

func sleepMilli(min int) {
	time.Sleep(time.Millisecond * time.Duration(min+rand.Intn(100)))
}

func startSpan(w http.ResponseWriter, r *http.Request) opentracing.Span {
	var span opentracing.Span
	operationName := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
	// get the context from the headers
	spanContext, err := opentracing.GlobalTracer().Extract(opentracing.TextMap,
		opentracing.HTTPHeadersCarrier(r.Header))
	if err == nil {
		// join the trace
		span = opentracing.StartSpan(operationName, opentracing.ChildOf(spanContext))
		span.LogEventWithPayload("join span", operationName)
	} else {
		// or start a new span
		span = opentracing.StartSpan(operationName)
		span.LogEventWithPayload("start span", operationName)
	}

	// set tags
	ext.HTTPMethod.Set(span, r.Method)
	ext.HTTPUrl.Set(span, r.URL.Path)
	return span
}

func tagAndLogError(span opentracing.Span, err error) {
	if span == nil || err == nil {
		return
	}
	log.Printf("call failed (%v)\n", err)
	// tag the span as errored
	ext.Error.Set(span, true)
	// log the error
	span.LogEventWithPayload("service error", err)

}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	filePath := "static/index.html"
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Printf("read file[%s] failed: %s\n", filePath, err.Error())
		w.WriteHeader(404)
		w.Write([]byte("404 Something went wrong - " + http.StatusText(404)))
	}
	w.Write(data)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Request start..."))
	span := startSpan(w, r)
	defer span.Finish()

	asyncReq, _ := http.NewRequest("GET", "http://localhost:8080/async", nil)
	// inject the trace information into the HTTP headers
	if err := span.Tracer().Inject(span.Context(),
		opentracing.TextMap,
		opentracing.HTTPHeadersCarrier(asyncReq.Header)); err != nil {
		log.Fatalf("%s: Couldn't inject headers (%v)\n", r.URL.Path, err)
	}

	go func() {
		sleepMilli(50)
		if _, err := http.DefaultClient.Do(asyncReq); err != nil {
			tagAndLogError(span, err)
		}
	}()

	sleepMilli(10)

	syncReq, _ := http.NewRequest("GET", "http://localhost:8080/service", nil)
	if err := span.Tracer().Inject(span.Context(),
		opentracing.TextMap,
		opentracing.HTTPHeadersCarrier(syncReq.Header)); err != nil {
		log.Fatalf("%s: Couldn't inject headers (%v)\n", r.URL.Path, err)
	}

	if _, err := http.DefaultClient.Do(syncReq); err != nil {
		tagAndLogError(span, err)
		return
	}
	w.Write([]byte("done!"))
}

func serviceHandler(w http.ResponseWriter, r *http.Request) {
	span := startSpan(w, r)
	defer span.Finish()

	sleepMilli(50)

	dbReq, _ := http.NewRequest("GET", "http://localhost:8080/db", nil)
	if err := span.Tracer().Inject(span.Context(),
		opentracing.TextMap,
		opentracing.HTTPHeadersCarrier(dbReq.Header)); err != nil {
		log.Fatalf("%s: Couldn't inject headers (%v)\n", r.URL.Path, err)
	}

	if _, err := http.DefaultClient.Do(dbReq); err != nil {
		tagAndLogError(span, err)
	}
}

func dbHandler(w http.ResponseWriter, r *http.Request) {
	span := startSpan(w, r)
	defer span.Finish()

	sleepMilli(25)
}

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
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/home", homeHandler)
	mux.HandleFunc("/async", serviceHandler)
	mux.HandleFunc("/service", serviceHandler)
	mux.HandleFunc("/db", dbHandler)
	fmt.Printf("serve %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
