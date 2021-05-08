package main

import (
	"context"
	"fmt"
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/propagation/b3"
	httpReporter "github.com/openzipkin/zipkin-go/reporter/http"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"trace/common"
)

func main() {
	reporter := httpReporter.NewReporter(common.ZipkinHttpReportHost)
	defer reporter.Close()

	endpoint, err := zipkin.NewEndpoint("test-cli", "localhost:5999")
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
	span := tracer.StartSpan("cli-request", zipkin.Kind(model.Client))
	defer span.Finish()

	request, err := http.NewRequest(http.MethodGet, "http://localhost:1501/", nil)
	if err != nil {
		span.Tag(string(zipkin.TagError), err.Error())
	}

	err = b3.InjectHTTP(request)(span.Context())
	if err != nil {
		span.Tag(string(zipkin.TagError), err.Error())
		log.Fatalln(err)
	}
	fmt.Println(request.Header)
	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		span.Tag(string(zipkin.TagError), err.Error())
		log.Fatalln(err)
	}

	childSpan, _ := tracer.StartSpanFromContext(zipkin.NewContext(context.Background(), span), "cli-resp")
	defer childSpan.Finish()
	childSpan.Tag(string(zipkin.TagHTTPStatusCode), strconv.Itoa(resp.StatusCode))
	content, err := ioutil.ReadAll(resp.Body)
	fmt.Println(string(content))
}