package main

import (
	"fmt"
	"github.com/opentracing/opentracing-go"
	opentracingLog "github.com/opentracing/opentracing-go/log"
	opentrcingZipkinImpl "github.com/openzipkin-contrib/zipkin-go-opentracing"
	"github.com/openzipkin/zipkin-go"
	httpReporter "github.com/openzipkin/zipkin-go/reporter/http"
	"io/ioutil"
	"log"
	"net/http"
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

	globalTracer := opentrcingZipkinImpl.Wrap(tracer)
	opentracing.SetGlobalTracer(globalTracer)

	span := opentracing.StartSpan("cli-request")
	defer span.Finish()

	request, err := http.NewRequest(http.MethodGet, "http://localhost:6063/", nil)
	if err != nil {
		span.SetTag("error",  err.Error())
		span.LogFields(opentracingLog.Error(err))
	}

	err = opentracing.GlobalTracer().Inject(
		span.Context(),
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(request.Header),
	)

	fmt.Println(request.Header)

	if err != nil {
		span.SetTag("error", err.Error())
		log.Fatalln(err)
	}

	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		span.SetTag("error", err.Error())
		log.Fatalln(err)
	}

	childSpan := opentracing.StartSpan("request-finish", opentracing.ChildOf(span.Context()))
	defer childSpan.Finish()

	childSpan.LogFields(opentracingLog.Int("Status code", resp.StatusCode))
	content, err := ioutil.ReadAll(resp.Body)

	fmt.Println(string(content))
}