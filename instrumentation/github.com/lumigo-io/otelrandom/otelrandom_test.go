package otelrandom_test

import (
	"context"
	"encoding/json"
	"github.com/adamluzsi/otelkit"
	"github.com/adamluzsi/testcase"
	"github.com/adamluzsi/testcase/assert"
	"github.com/adamluzsi/testcase/random"
	"github.com/shirou/gopsutil/v3/cpu"
	"go.opentelemetry.io/contrib/instrumentation/github.com/lumigo-io/otelrandom"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"testing"
	"time"
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
		otelstub                    = otelkit.Stub(t)
		randomGenerator             = StubRandomGenerator{}
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
		otelstub                    = otelkit.Stub(t)
		ctx                         = context.Background()
		inputArgument               = rnd.IntBetween(1, 42)
		randomGenerator             = StubRandomGenerator{}
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
		otelstub                    = otelkit.Stub(t)
		ctx                         = context.Background()
		randomGenerator             = StubRandomGenerator{}
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
		randomGenerator = StubRandomGenerator{}
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

func TestRandomGeneratorInstrument_Intn_asyncSpanExample(t *testing.T) {
	var (
		otelstub        = otelkit.Stub(t)
		randomGenerator = StubRandomGenerator{}
		subject         = otelrandom.NewRandomGenerator(randomGenerator)
	)

	rootCtx, rootSpan := otel.GetTracerProvider().Tracer("TestTracer").Start(context.Background(), "RootSpan")
	defer rootSpan.End()

	_ = subject.Intn(rootCtx, rnd.IntB(1, 42))

	t.Log("eventually, we a span will be exported that has a link pointing to an already exported span")
	assert.EventuallyWithin(3*time.Second).Assert(t, func(t assert.It) {
		assert.OneOf(t, otelstub.SpanExporter.ExportedSpans(), func(t assert.It, span sdktrace.ReadOnlySpan) {
			assert.Equal(t, rootSpan.SpanContext(), span.Parent(),
				"expected to have the root span as its traceparent and not Intn method span")

			assert.OneOf(t, span.Links(), func(t assert.It, link sdktrace.Link) { // one of the span's link is ...
				assert.NotEqual(t, link.SpanContext, span.Parent(),
					"expected that the linked span is NOT the traceparent")

				assert.OneOf(t, otelstub.SpanExporter.ExportedSpans(), func(t assert.It, exportedSpan sdktrace.ReadOnlySpan) {

					assert.Equal(t, link.SpanContext, exportedSpan.SpanContext(),
						"expected that the linked span is also a valid and existing exported span")

				})
			})
		})
	})
}

func TestRandomGeneratorInstrument_Intn_raceSafe(t *testing.T) {
	var (
		otelstub        = otelkit.Stub(t)
		randomGenerator = StubRandomGenerator{}
		subject         = otelrandom.NewRandomGenerator(randomGenerator)
		ctx             = context.Background()
	)
	testcase.Race(func() { // will create a arrangement where unsafe code will have race condition
		subject.Intn(ctx, rnd.IntB(1, 42))
	}, func() {
		subject.Intn(ctx, rnd.IntB(1, 42))
	}, func() {
		assert.EventuallyWithin(time.Second).Assert(t, func(it assert.It) {
			it.Must.NotEmpty(otelstub.SpanExporter.ExportedSpans())
		})
	})
}

func TestRandomGeneratorInstrument_Intn_contextPropagation(t *testing.T) {
	const ctxValKey = "key"
	var (
		_             = otelkit.Stub(t)
		inputArgument = rnd.IntB(1, 42)
		inputContext  = context.WithValue(context.Background(), ctxValKey, "value")
		randomGen     = StubRandomGenerator{IntnFunc: func(ctx context.Context, n int) int {
			assert.NotNil(t, ctx)
			assert.Equal(t, ctx.Value(ctxValKey), inputContext.Value(ctxValKey),
				"expected that the received context has the received context values")
			assert.NotNil(t, trace.SpanFromContext(ctx),
				"expected that the propagated context has a span already")

			assert.Equal(t, inputArgument, n)
			return rnd.IntN(n)
		}}
		instrumentedRandomGenerator = otelrandom.NewRandomGenerator(randomGen) // use global otel trace provider
	)
	_ = instrumentedRandomGenerator.Intn(inputContext, inputArgument)
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
	if stub.IntnFunc == nil {
		return rnd.IntN(n)
	}
	return stub.IntnFunc(ctx, n)
}
