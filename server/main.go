package main

import (
	"context"
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/propagation/b3"
	httpReporter "github.com/openzipkin/zipkin-go/reporter/http"
	"log"
	"math/rand"
	"net/http"
	"time"
	"trace/common"
)

func main() {
	reporter := httpReporter.NewReporter(common.ZipkinHttpReportHost)
	defer reporter.Close()

	endpoint, err := zipkin.NewEndpoint("test-server", "localhost:0501")
	if err != nil {
		log.Fatalln(err)
	}

	tracer, err := zipkin.NewTracer(
		reporter,
		zipkin.WithLocalEndpoint(endpoint),
		zipkin.WithSampler(zipkin.AlwaysSample),
		)

	if err != nil {
		log.Fatalln(err)
	}

	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		rootContext := tracer.Extract(b3.ExtractHTTP(request))
		if rootContext.Err != nil {
			log.Fatalln(err)
		}
		span := tracer.StartSpan("test-serv", zipkin.Parent(rootContext))
		childSpan, childSpanContext := tracer.StartSpanFromContext(zipkin.NewContext(context.Background(), span), "db-query")
		defer span.Finish()
		defer childSpan.Finish()

		time.Sleep(time.Millisecond * 5)
		remoteEndpoint, err := zipkin.NewEndpoint("db-mysql", "localhost:3306")
		if err != nil {
			log.Fatalln(err)
		}

		childSpan.SetRemoteEndpoint(remoteEndpoint)
		childSpan.Annotate(time.Now(), "query-start")
		time.Sleep(time.Duration(rand.Intn(3)))
		childSpan.Annotate(time.Now(), "query-end")
		childSpan.Tag(string(zipkin.TagError), "query-failed")

		subChildSpan, _ := tracer.StartSpanFromContext(childSpanContext, "assemble-db-query-result")
		defer subChildSpan.Finish()
		time.Sleep(time.Second * 2)

		writer.Write([]byte("hello world"))
	})

	if err = http.ListenAndServe(":9501", nil); err != nil {
		log.Fatalln(err)
	}
}
