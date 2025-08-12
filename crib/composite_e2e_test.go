package crib_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
)

type (
	// ErrScalar doesn't include an Apply method. So cannot be used as a Component.
	ErrScalar struct{}

	SimpleProducer struct {
		Arg any
	}

	SimpleResult struct {
		Arg any
	}

	SimpleConsumer struct{}
)

func NewErrScaler() *ErrScalar {
	return &ErrScalar{}
}

func (*ErrScalar) String() string {
	return "sdk.composite.ErrScalar"
}

func NewSimpleProducer(arg any) func() *SimpleProducer {
	return func() *SimpleProducer {
		return &SimpleProducer{
			Arg: arg,
		}
	}
}

func (s *SimpleProducer) Apply() *SimpleResult {
	fmt.Printf("[SimpleProduce] Producing *SimpleResult from Apply with arg %v\n", s.Arg)
	return &SimpleResult{
		Arg: s.Arg,
	}
}

func (*SimpleProducer) String() string {
	return "sdk.composite.SimpleProducer"
}

func NewSimpleConsumer() *SimpleConsumer {
	return &SimpleConsumer{}
}

func (*SimpleConsumer) Apply(ctx context.Context, res *SimpleResult) {
	fmt.Printf("[SimpleConsumer] Consuming result %v from SimpleConsumer with context %v\n", res, spew.Sprintf("%v", ctx))
}

func (*SimpleConsumer) String() string {
	return "sdk.composite.SimpleConsumer"
}

func TestCompositeErrRegistration(t *testing.T) {
	ctx := t.Context()
	app := crib.NewTestApp(t)
	ctx = internal.ContextWithConstruct(ctx, app.Chart)

	composite := crib.NewComposite(
		NewErrScaler,
		NewErrScaler,
		NewSimpleProducer,
	)

	res, err := composite(ctx)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "sdk.composite.ErrScalar")
	assert.ErrorContains(t, err, "missing Apply method")
	assert.ErrorContains(t, err, ".SimpleProducer")
	assert.ErrorContains(t, err, "with non-zero required arguments")
	assert.Nil(t, res)
	t.Logf("Got error: %v", err)
}

func TestCompositeE2E(t *testing.T) {
	ctx := t.Context()
	app := crib.NewTestApp(t)
	ctx = internal.ContextWithConstruct(ctx, app.Chart)

	composite := crib.NewComposite(
		NewSimpleProducer("Hello, World!"),
		NewSimpleConsumer,
	)

	res, err := composite(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Implements(t, (*crib.Component)(nil), res)
}
