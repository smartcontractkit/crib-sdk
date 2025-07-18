package clientsideapply

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/crib-sdk/internal/adapter/mempools"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
)

func TestEchoExecute(t *testing.T) {
	t.Parallel()

	input := &domain.ClientSideApplyManifest{
		Spec: domain.ClientSideApplySpec{
			OnFailure: "abort",
			Action:    "cribctl",
			Args: []string{
				"apply",
				"plan",
				"example",
			},
		},
	}

	buf, reset := mempools.BytesBuffer.Get()
	t.Cleanup(reset)

	runner, err := NewEchoRunner(buf)
	require.NoError(t, err)

	result, err := runner.Execute(t.Context(), input)
	require.NoError(t, err)

	assert.Equal(t, result.Output, buf.Bytes())
	assert.Equal(t, "ClientSideApply:\n  OnFailure: abort\n  Action: cribctl\n  Args:\n    - apply\n    - plan\n    - example\n", string(result.Output))
}
