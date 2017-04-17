package server

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
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
		fmt.Printf("join: %v\n", r.Header)
		span = opentracing.StartSpan(operationName, opentracing.ChildOf(spanContext))
		span.LogEventWithPayload("join span", operationName)
	} else {
		// or start a new span
		fmt.Printf("start: %v\n", r.Header)
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

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	filePath := "../server/static/index.html"
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Printf("read file[%s] failed: %s\n", filePath, err.Error())
		w.WriteHeader(404)
		w.Write([]byte("404 Something went wrong - " + http.StatusText(404)))
	}
	w.Write(data)
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
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

func ServiceHandler(w http.ResponseWriter, r *http.Request) {
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

func DBHandler(w http.ResponseWriter, r *http.Request) {
	span := startSpan(w, r)
	defer span.Finish()

	sleepMilli(25)
}
