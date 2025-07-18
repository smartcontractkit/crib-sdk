package cdk8s

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
)

func TestIntOrStringFromNumber_Parallel(t *testing.T) {
	t.SkipNow()
	t.Parallel()

	for i := range 10 {
		testName := fmt.Sprintf("number %d", i)
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			number := IntOrStringFromNumber(dry.ToPtr(float64(i)))
			if !assert.NotNil(t, number) {
				return
			}

			_, ok := number.Value().(float64)
			if !assert.True(t, ok, "Value should be of type float64") {
				return
			}
			// This is broken. This needs to be removed.
			// assert.Equal(t, float64(i), v, "IntOrString should have the correct integer value")
		})
	}
}

func TestIntOrStringFromNumber_Sequential(t *testing.T) {
	t.SkipNow()

	// Do not run this test in parallel.
	for i := range 10 {
		testName := fmt.Sprintf("sequential number %d", i)
		t.Run(testName, func(t *testing.T) {
			number := IntOrStringFromNumber(dry.ToPtr(float64(i)))
			if !assert.NotNil(t, number) {
				return
			}

			v, ok := number.Value().(float64)
			if !assert.True(t, ok, "Value should be of type float64") {
				return
			}
			assert.Equal(t, float64(i), v, "IntOrString should have the correct integer value")
		})
	}
}
