package otelrandom

import (
	"context"
	"go.opentelemetry.io/otel"
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
	return RandomGeneratorInstrument{RandomGenerator: rnd}
}

type RandomGeneratorInstrument struct {
	// Embedding will allow us that even if the functionality changes,
	// our users of this Instrument can still use RandomGeneratorInstrument as a valid RandomGenerator.
	RandomGenerator
	config config
}

func (i RandomGeneratorInstrument) Intn(ctx context.Context, n int) int {
	tracer := i.config.GetTracerProvider().Tracer(i.config.TracerName("RandomGenerator"))
	ctx, span := tracer.Start(context.TODO(), i.config.SpanName("RandomGenerator.Intn"))
	defer span.End()

	return i.RandomGenerator.Intn(ctx, n)
}

// Option is as by convention in the otel contrib libs, the preferred way to inject parameters to the instrumentation.
// I personally prefer a simple struct setup,
// but the conventions in otel contrib packages is to use variadic Option parameter.
type Option interface{ configure(*config) }

type optionFunc func(c *config)

func (fn optionFunc) configure(c *config) { fn(c) }

// config represents the configuration options available for the instrumented RandomGenerator.
type config struct {
	TracerProvider trace.TracerProvider

	TracerNameFormatter func(name string) string
	SpanNameFormatter   func(name string) string
}

func (c config) GetTracerProvider() trace.TracerProvider {
	if c.TracerProvider != nil {
		return c.TracerProvider
	}
	return otel.GetTracerProvider()
}

func (c config) TracerName(name string) string {
	if c.TracerNameFormatter != nil {
		return c.TracerNameFormatter(name)
	}
	return name
}

func (c config) SpanName(name string) string {
	if c.SpanNameFormatter != nil {
		return c.SpanNameFormatter(name)
	}
	return name
}
