package clientsideapply

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
)

func TestCmdExecute(t *testing.T) {
	t.Parallel()

	input := &domain.ClientSideApplyManifest{
		Spec: domain.ClientSideApplySpec{
			OnFailure: "abort",
			Action:    "cmd",
			Args: []string{
				"for i in {1..5}; do echo \"Iteration $i\"; done",
			},
		},
	}

	runner, err := NewCmdRunner()
	require.NoError(t, err)

	result, err := runner.Execute(t.Context(), input)
	require.NoError(t, err)
	require.NotEmpty(t, result.Output, "Expected non-empty output from command execution")

	resStr := string(result.Output)
	assert.Contains(t, resStr, "Iteration", "Expected output to contain 'Iteration'")
	assert.Contains(t, resStr, "1", "Expected output to contain '1'")
}
