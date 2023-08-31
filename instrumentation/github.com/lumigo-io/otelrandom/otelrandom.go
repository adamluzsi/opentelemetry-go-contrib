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

const (
	tracerName   = "opentelemetry-go-contrib/instrumentation/github.com/lumigo-io/otelrandom/RandomGenerator"
	traceVersion = "0.0.1"
)

// RandomGenerator represents the function signature we've identified for instrumentation during our discussion.
//
// Much like how otelhttp leverages http.Handler as its foundational layer for embedding telemetry into an http.Handler,
// RandomGenerator serves a similar purpose in our API.
type RandomGenerator interface {
	Intn(ctx context.Context, n int) int
}

func NewRandomGenerator(rnd RandomGenerator, opts ...Option) RandomGenerator {
	instrument := randomGeneratorInstrument{RandomGenerator: rnd}
	for _, opt := range opts {
		opt.configure(&instrument.config)
	}
	if instrument.config.TracerProvider == nil {
		instrument.config.TracerProvider = otel.GetTracerProvider()
	}
	instrument.config.Tracer = instrument.config.TracerProvider.Tracer(tracerName,
		trace.WithInstrumentationVersion(traceVersion))
	return instrument
}

type randomGeneratorInstrument struct {
	// Embedding will allow us that even if the functionality changes,
	// our users of this Instrument can still use randomGeneratorInstrument as a valid RandomGenerator.
	RandomGenerator
	config config
}

func (i randomGeneratorInstrument) Intn(ctx context.Context, n int) int {
	spanCtx, span := i.config.Tracer.Start(ctx, "RandomGenerator.Intn")
	defer span.End()
	span.SetAttributes(i.payloadAttributes(n)...)
	span.SetAttributes(i.profilingAttributes()...)
	// Passing the root context ensure the expected structure from the specification.
	// passing current span's span context to ensure linking
	go i.exampleAsyncSpan(ctx, span.SpanContext())
	return i.RandomGenerator.Intn(spanCtx, n) // spanCtx passed to link possible further sub span creations
}

func (i randomGeneratorInstrument) payloadAttributes(payload any) []attribute.KeyValue {
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

func (i randomGeneratorInstrument) profilingAttributes() []attribute.KeyValue {
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

// exampleAsyncSpan is expected to executed with the go keyword, to simulate async workload.
// trace.Span created as part of the function
// is what is often considered as async span in the Go OpenTelemetry implementation.
func (i randomGeneratorInstrument) exampleAsyncSpan(
	// rootContext must be the original context.Context that might or might not contain the root span.
	// It must not contain the Intn method's span context
	rootContext context.Context,
	// spanToLink is to which we will link in our span.
	// It must be the Intn method's span context.
	spanToLink trace.SpanContext,
) {
	_, asyncSpan := i.config.Tracer.Start(rootContext, "AsyncSpan",
		trace.WithLinks(trace.Link{SpanContext: spanToLink}))
	defer asyncSpan.End()
	//
	// some work to do
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
	Tracer         trace.Tracer
}
