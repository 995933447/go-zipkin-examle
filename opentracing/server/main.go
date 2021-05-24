package main

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	opentracingLog "github.com/opentracing/opentracing-go/log"
	opentrcingZipkinImpl "github.com/openzipkin-contrib/zipkin-go-opentracing"
	"github.com/openzipkin/zipkin-go"
	httpReporter "github.com/openzipkin/zipkin-go/reporter/http"
	"log"
	"math/rand"
	"time"
	"trace/common"
)

func main()  {
	reporter := httpReporter.NewReporter(common.ZipkinHttpReportHost)
	localEndpoint, err := zipkin.NewEndpoint("opentracing-sev-test", "localhost:9502")
	if err != nil {
		log.Fatalln(err)
	}

	tracer, err := zipkin.NewTracer(
		reporter,
		zipkin.WithSampler(zipkin.AlwaysSample),
		zipkin.WithLocalEndpoint(localEndpoint),
		)
	if err != nil {
		log.Fatalln(err)
	}

	globalTracer := opentrcingZipkinImpl.Wrap(tracer)
	opentracing.SetGlobalTracer(globalTracer)

	defaultRouter := gin.New()
	defaultRouter.GET("/", func(context *gin.Context) {
		parentContext, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(context.Request.Header))
		var span opentracing.Span
		if err == nil {
			span = opentracing.StartSpan("test-opentrace-server", opentracing.ChildOf(parentContext))
		} else {
			span = opentracing.StartSpan("test-opentrace-server")
		}
		defer span.Finish()

		span.SetTag("db-mysql", "localhost:3306")
		time.Sleep(time.Millisecond * 5)
		span.LogFields(opentracingLog.Int64("query-start", time.Now().Unix()))
		time.Sleep(time.Duration(rand.Intn(3)))
		span.LogFields(opentracingLog.Int64("query-end", time.Now().Unix()))

		span.LogFields(opentracingLog.Error(errors.New("Msql query failed.")))
		span.SetTag("error", "Msql query failed tag.")
		context.String(200, "Ok.")
	})
	defaultRouter.Run(":6063")
}