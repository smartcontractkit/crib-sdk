package internal

import "sync"

// JSIIKernelMutex is global mutex to synchronize all parallel invocations of jsii kernel
// This is required to be used in Tests when test code contains any invocations to jsii kernel and uses t.Parallel()
// It can be also used in the productions code if it contains any parallel invocations.
var JSIIKernelMutex sync.Mutex
