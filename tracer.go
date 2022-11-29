package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	tracer = otel.GetTracerProvider().Tracer(
		"instrumentationName",
		trace.WithInstrumentationVersion("v0.1.0"),
		trace.WithSchemaURL(semconv.SchemaURL),
	)
)

func setupOtel(ctx context.Context) func(context.Context) error {
	client := otlptracehttp.NewClient(otlptracehttp.WithInsecure())
	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		log.Fatal(fmt.Errorf("creating OTLP trace exporter: %w", err))
	}
	consoleExporter, err := newExporter(os.Stdout)
	if err != nil {
		log.Fatal(fmt.Errorf("creating stdout trace exporter: %w", err))
	}
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(consoleExporter),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(newResource()),
	)
	otel.SetTracerProvider(tracerProvider)

	return tracerProvider.Shutdown
}

func newExporter(w io.Writer) (sdktrace.SpanExporter, error) {
	return stdouttrace.New(
		stdouttrace.WithWriter(w),
		//		stdouttrace.WithPrettyPrint(),
		stdouttrace.WithoutTimestamps(),
	)
}
func newResource() *resource.Resource {
	r, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("importer"),
			semconv.ServiceVersionKey.String("v0.1.0"),
			attribute.String("environment", "demo"),
		),
	)
	return r
}
