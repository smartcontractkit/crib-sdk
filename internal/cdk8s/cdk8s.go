package cdk8s

import (
	"github.com/cdk8s-team/cdk8s-plus-go/cdk8splus30/v2/k8s"

	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
)

// IntOrStringFromNumber is wrapper for cdk8s IntOrStringFromNumber, which doesn't work in parallel calls
// the wrapper uses global mutex to synchronize all calls to the cdk8s IntOrStringFromNumber.
func IntOrStringFromNumber(value *float64) (res k8s.IntOrString) {
	defer func() {
		if r := recover(); r != nil {
			// If cdk8s IntOrStringFromNumber panics, we return a nil IntOrString.
			res = dry.Empty[k8s.IntOrString]()
		}
	}()

	// cdk8s has some level of globally shared state, so we need to acquire a lock.
	internal.JSIIKernelMutex.Lock()
	defer internal.JSIIKernelMutex.Unlock()

	return k8s.IntOrString_FromNumber(value)
}
