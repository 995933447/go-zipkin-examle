package main

import (
	"context"
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/propagation/b3"
	httpReporter "github.com/openzipkin/zipkin-go/reporter/http"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
	"trace/common"
)

func main() {
	reporter := httpReporter.NewReporter(common.ZipkinHttpReportHost)
	defer reporter.Close()

	hostPort, _ := os.Hostname()
	endpoint, err := zipkin.NewEndpoint("test-server", hostPort)
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
		// 从当前Http header获取客户端spanContext
		rootContext := tracer.Extract(b3.ExtractHTTP(request))
		if rootContext.Err != nil {
			log.Fatalln(err)
		}

		// 依据客户端span为parent开启一个子span,如果传入extract出来的rootContext为空的spanContext会开启一个新的根span
		span := tracer.StartSpan("test-serv", zipkin.Parent(rootContext))
		// 依据当前span开启一个子span,模拟数据库查询
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

		// 模拟查询失败
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
