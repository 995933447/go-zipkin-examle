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
	// span可以指定种类,当前当前跨度场景,如CLIENT, SERVER, PRODUCER, CONSUMER等
	span := tracer.StartSpan("cli-request", zipkin.Kind(model.Client))
	defer span.Flush()

	request, err := http.NewRequest(http.MethodGet, "http://localhost:1501/", nil)
	if err != nil {
		span.Tag(string(zipkin.TagError), err.Error())
	}

	// 将当前spanContext注入到http请求头中,完成span在同一链路中的传递
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
	defer childSpan.Flush()

	childSpan.Tag(string(zipkin.TagHTTPStatusCode), strconv.Itoa(resp.StatusCode))
	content, err := ioutil.ReadAll(resp.Body)

	fmt.Println(string(content))
}