package otelrandom_test

import (
	"context"
	"encoding/json"
	"github.com/adamluzsi/opentelemetry-go-contrib/instrumentation/github.com/lumigo-io/otelrandom"
	"github.com/adamluzsi/otelkit"
	"github.com/adamluzsi/testcase/assert"
	"github.com/adamluzsi/testcase/random"
	"github.com/shirou/gopsutil/v3/cpu"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"testing"
)

// I use blackbox testing, to gain a better understanding about my own API
// by forcing my perspective to be a user of my own api during the testing scenarios.

var rnd = random.New(random.CryptoSeed{})

func TestRandomGeneratorInstrument_Intn_smoke(t *testing.T) {
	var (
		otelstub         = otelkit.Stub(t)
		expOutputValue   = rnd.Int()
		expInputArgument = rnd.IntB(1, 42)
		randomGen        = StubRandomGenerator{IntnFunc: func(ctx context.Context, n int) int {
			assert.Equal(t, expInputArgument, n)
			return expOutputValue
		}}
		instrumentedRandomGenerator = otelrandom.NewRandomGenerator(randomGen) // use global otel trace provider
	)

	got := instrumentedRandomGenerator.Intn(context.Background(), expInputArgument)
	assert.Equal(t, expOutputValue, got)
	assert.NotEmpty(t, otelstub.SpanExporter.ExportedSpans(), "expected that some span export actually happened")
}

func TestRandomGeneratorInstrument_Intn_useTraceparent(t *testing.T) {
	var (
		otelstub        = otelkit.Stub(t)
		randomGenerator = StubRandomGenerator{IntnFunc: func(ctx context.Context, n int) int {
			return rnd.IntN(n)
		}}
		instrumentedRandomGenerator = otelrandom.NewRandomGenerator(randomGenerator) // use global otel trace provider
	)

	ctx, rootSpan := otel.GetTracerProvider().Tracer("RootTracer").
		Start(context.Background(), "RootSpan")
	defer rootSpan.End()

	_ = instrumentedRandomGenerator.Intn(ctx, 5)
	assert.NotEmpty(t, otelstub.SpanExporter.ExportedSpans())
	assert.OneOf(t, otelstub.SpanExporter.ExportedSpans(), func(t assert.It, got sdktrace.ReadOnlySpan) {
		t.Log("the new span has the trace parent id")
		parent := got.Parent()
		assert.NotEmpty(t, parent)
		assert.Equal(t, rootSpan.SpanContext().TraceID(), parent.TraceID())
	})
}

func TestRandomGeneratorInstrument_Intn_hasLumigoPayloadAttributes(t *testing.T) {
	var (
		otelstub        = otelkit.Stub(t)
		ctx             = context.Background()
		inputArgument   = rnd.IntBetween(1, 42)
		randomGenerator = StubRandomGenerator{IntnFunc: func(ctx context.Context, n int) int {
			return rnd.IntN(n)
		}}
		instrumentedRandomGenerator = otelrandom.NewRandomGenerator(randomGenerator) // use global otel trace provider
	)

	payloadValue, err := json.Marshal(inputArgument)
	assert.NoError(t, err)

	expectedAttributes := []attribute.KeyValue{
		otelrandom.Payload.String(string(payloadValue)),
		otelrandom.PayloadSize.Int(len(payloadValue)),
	}

	_ = instrumentedRandomGenerator.Intn(ctx, inputArgument)

	assert.OneOf(t, otelstub.SpanExporter.ExportedSpans(), func(t assert.It, got sdktrace.ReadOnlySpan) {
		assert.Contain(t, got.Attributes(), expectedAttributes)
	})
}

func TestRandomGeneratorInstrument_Intn_withCPUTimeAttribute(t *testing.T) {
	var (
		otelstub        = otelkit.Stub(t)
		ctx             = context.Background()
		randomGenerator = StubRandomGenerator{IntnFunc: func(ctx context.Context, n int) int {
			return rnd.IntN(n)
		}}
		instrumentedRandomGenerator = otelrandom.NewRandomGenerator(randomGenerator) // use global otel trace provider
	)

	totalCPUTime := getCurrentCPUTime(t)

	_ = instrumentedRandomGenerator.Intn(ctx, rnd.IntB(1, 42))

	assert.OneOf(t, otelstub.SpanExporter.ExportedSpans(), func(t assert.It, got sdktrace.ReadOnlySpan) {
		assert.OneOf(t, got.Attributes(), func(t assert.It, got attribute.KeyValue) {
			assert.Equal(t, got.Key, otelrandom.ProcessCPUTime)
			assert.NotEmpty(t, got.Value)
			assert.Equal(t, attribute.FLOAT64, got.Value.Type())
			gotTotal := got.Value.AsFloat64()
			assert.True(t, totalCPUTime <= gotTotal)
		})
	})
}

func TestRandomGeneratorInstrument_Intn_withTracerProvider(t *testing.T) {
	var (
		otelstub        = otelkit.Stub(t)
		ctx             = context.Background()
		randomGenerator = StubRandomGenerator{IntnFunc: func(ctx context.Context, n int) int {
			return rnd.IntN(n)
		}}
	)

	t.Log("given we set the global trace provider to a NoOperationTracerProvider")
	t.Log("if it is being used, then it will be ignored")
	ogTP := otel.GetTracerProvider()
	otel.SetTracerProvider(trace.NewNoopTracerProvider())
	t.Cleanup(func() { otel.SetTracerProvider(ogTP) })

	instrumentedRandomGenerator := otelrandom.NewRandomGenerator(randomGenerator,
		otelrandom.WithTracerProvider(otelstub.TracerProvider))

	assert.Empty(t, otelstub.SpanExporter.ExportedSpans())
	_ = instrumentedRandomGenerator.Intn(ctx, rnd.IntB(1, 42))
	assert.NotEmpty(t, otelstub.SpanExporter.ExportedSpans())
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func getCurrentCPUTime(tb testing.TB) float64 {
	cpuTimes, err := cpu.Times(false)
	assert.NoError(tb, err)
	var total float64
	for _, cpuTime := range cpuTimes {
		total += cpuTime.User + cpuTime.System
	}
	return total
}

type StubRandomGenerator struct {
	IntnFunc func(ctx context.Context, n int) int
}

func (stub StubRandomGenerator) Intn(ctx context.Context, n int) int {
	return stub.IntnFunc(ctx, n)
}
