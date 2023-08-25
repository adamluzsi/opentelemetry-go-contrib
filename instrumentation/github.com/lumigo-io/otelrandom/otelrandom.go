package otelrandom

import (
	"context"
	"encoding/json"
	"github.com/shirou/gopsutil/v3/cpu"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	// Payload is the JSON serialised content of the captured input payload
	Payload = attribute.Key("method.payload")
	// PayloadSize is the Payload bytes size
	PayloadSize = attribute.Key("method.payload.size")
	// ProcessCPUTime is the total cpu time for the current process.
	ProcessCPUTime = attribute.Key("process.cpu.time")
)

// RandomGenerator represents the function signature we've identified for instrumentation during our discussion.
//
// Much like how otelhttp leverages http.Handler as its foundational layer for embedding telemetry into an http.Handler,
// RandomGenerator serves a similar purpose in our API.
type RandomGenerator interface {
	Intn(ctx context.Context, n int) int
}

func NewRandomGenerator(rnd RandomGenerator, opts ...Option) RandomGenerator {
	instrumentedRandomGenerator := RandomGeneratorInstrument{RandomGenerator: rnd}
	for _, opt := range opts {
		opt.configure(&instrumentedRandomGenerator.config)
	}
	return instrumentedRandomGenerator
}

type RandomGeneratorInstrument struct {
	// Embedding will allow us that even if the functionality changes,
	// our users of this Instrument can still use RandomGeneratorInstrument as a valid RandomGenerator.
	RandomGenerator
	config config
}

func (i RandomGeneratorInstrument) Intn(ctx context.Context, n int) int {
	tracer := i.config.GetTracerProvider().Tracer(i.config.TracerName("RandomGenerator"))
	ctx, span := tracer.Start(ctx, i.config.SpanName("RandomGenerator.Intn"))
	defer span.End()
	span.SetAttributes(i.payloadAttributes(n)...)
	span.SetAttributes(i.profilingAttributes()...)
	return i.RandomGenerator.Intn(ctx, n)
}

func (i RandomGeneratorInstrument) payloadAttributes(payload any) []attribute.KeyValue {
	payloadValue, err := json.Marshal(payload)
	if err != nil {
		// nice to have could be to add logging here,
		// but because we are in an instrument scope,
		// and we don't own the application logging configuration
		// we use a testing based approach to ensure no unserializable value is passed to this method.
		return nil
	}
	return []attribute.KeyValue{
		Payload.String(string(payloadValue)),
		PayloadSize.Int(len(payloadValue)),
	}
}

func (i RandomGeneratorInstrument) profilingAttributes() []attribute.KeyValue {
	times, err := cpu.Times(false /* false -> aggregate total */)
	if err != nil {
		return nil // log error
	}
	var total float64
	for _, tm := range times {
		total += tm.User + tm.System
	}
	return []attribute.KeyValue{
		ProcessCPUTime.Float64(total),
	}
}

// Option is as by convention in the otel contrib libs, the preferred way to inject parameters to the instrumentation.
// I personally prefer a simple struct setup,
// but the conventions in otel contrib packages is to use variadic Option parameter.
type Option interface{ configure(*config) }

func WithTracerProvider(tp trace.TracerProvider) Option {
	return optionFunc(func(c *config) { c.TracerProvider = tp })
}

type optionFunc func(c *config)

func (fn optionFunc) configure(c *config) { fn(c) }

// config represents the configuration options available for the instrumented RandomGenerator.
// In other instrument libraries, this is the convention rather than having these fields on the instrument itself.
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
