package otelrandom

import (
	"context"
	"go.opentelemetry.io/otel/trace"
)

// RandomGenerator represents the function signature we've identified for instrumentation during our discussion.
//
// Much like how otelhttp leverages http.Handler as its foundational layer for embedding telemetry into an http.Handler,
// RandomGenerator serves a similar purpose in our API.
type RandomGenerator interface {
	Intn(ctx context.Context, n int) int
}

func NewRandomGenerator(rnd RandomGenerator, opts ...Option) RandomGenerator {
	return nil
}

// Option is as by convention in the otel contrib libs, the preferred way to inject parameters to the instrumentation.
// I personally prefer a simple struct setup,
// but the conventions in otel contrib packages is to use variadic Option parameter.
type Option interface{ configure(*config) }

type optionFunc func(c *config)

func (fn optionFunc) configure(c *config) { fn(c) }

// config represents the configuration options available for the instrumented RandomGenerator.
type config struct {
	TracerProvider   trace.TracerProvider
	Tracer           trace.Tracer
	SpanStartOptions []trace.SpanStartOption
}
