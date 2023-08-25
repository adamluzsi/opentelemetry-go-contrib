package otelrandom_test

import (
	"context"
	"github.com/adamluzsi/opentelemetry-go-contrib/instrumentation/github.com/lumigo-io/otelrandom"
	"github.com/adamluzsi/otelkit"
	"github.com/adamluzsi/testcase/assert"
	"github.com/adamluzsi/testcase/random"
	"testing"
)

// I use blackbox testing, to gain a better understanding about my own API
// by forcing my perspective to be a user of my own api during the testing scenarios.

var rnd = random.New(random.CryptoSeed{})

func TestRandomGeneratorInstrument_Intn_smoke(t *testing.T) {
	var (
		otelstub         = otelkit.Stub(t)
		expOutputValue   = rnd.Int()
		expInputArgument = rnd.Int()
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

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type StubRandomGenerator struct {
	IntnFunc func(ctx context.Context, n int) int
}

func (stub StubRandomGenerator) Intn(ctx context.Context, n int) int {
	return stub.IntnFunc(ctx, n)
}
