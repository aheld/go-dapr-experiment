package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	dapr "github.com/dapr/go-sdk/client"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	_ "time/tzdata"
)

type Message struct {
	Source string `json:"source"`
	Input  string `json:"input"`
}

const PUBSUB_NAME = "inputpubsub"
const PUBSUB_TOPIC = "inputs"

func main() {
	ctx := context.Background()
	shutdown := setupOtel(ctx)
	defer func() {
		if err := shutdown(ctx); err != nil {
			log.Fatal(err)
		}
	}()

	ctx, span := tracer.Start(ctx, "Start")
	defer span.End()

	msg := Message{"Dapr Publish", "hello"}
	daprPublish(ctx, &msg)

	msg2 := Message{"HTTP Publish", "hi there"}
	publish(ctx, &msg2)
}

func daprPublish(ctx context.Context, msg *Message) {
	ctx, span := tracer.Start(ctx, "Publish Func (dapr)")
	defer span.End()
	client, err := dapr.NewClient()
	if err != nil {
		panic(err)
	}
	defer client.Close()

	fmt.Println("TraceID: ", span.SpanContext().TraceID().String())
	ctx = client.WithTraceID(ctx, span.SpanContext().TraceID().String())
	msg.Input = ""
	if err := client.PublishEvent(ctx, PUBSUB_NAME, PUBSUB_TOPIC, msg); err != nil {
		panic(err)
	}

	span.AddEvent("Published Input",
		trace.WithAttributes(attribute.String("Source", msg.Source),
			attribute.String("Input", msg.Input)),
	)

	fmt.Println("Published data for:", msg.Source)
	fmt.Println("\n*******\n")

}

func publish(ctx context.Context, msg *Message) {
	ctx, span := tracer.Start(ctx, "Publish Func")
	defer span.End()

	var DAPR_HOST, DAPR_HTTP_PORT string
	var okHost, okPort bool
	if DAPR_HOST, okHost = os.LookupEnv("DAPR_HOST"); !okHost {
		DAPR_HOST = "http://localhost"
	}
	if DAPR_HTTP_PORT, okPort = os.LookupEnv("DAPR_HTTP_PORT"); !okPort {
		DAPR_HTTP_PORT = "3500"
	}

	msgString, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	client := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}
	req, err := http.NewRequestWithContext(ctx, "POST", DAPR_HOST+":"+DAPR_HTTP_PORT+"/v1.0/publish/"+PUBSUB_NAME+"/"+PUBSUB_TOPIC, bytes.NewBuffer(msgString))
	if err != nil {
		span.RecordError(err)
		log.Fatal(err.Error())
		os.Exit(1)
	}

	// Publish an event using Dapr pub/sub
	if _, err = client.Do(req); err != nil {
		span.RecordError(err)
		log.Fatal(err)
	}
	span.AddEvent("Published Input",
		trace.WithAttributes(attribute.String("Source", msg.Source),
			attribute.String("Input", msg.Input)),
	)

	fmt.Println("Published data for:", msg.Source)
	fmt.Println("\n*******\n")

}
