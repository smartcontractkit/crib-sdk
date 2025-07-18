package clientsideapply

import (
	"context"
	"strings"

	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
)

func (w *wrappedRunner) Execute(ctx context.Context, input *domain.ClientSideApplyManifest) (*domain.RunnerResult, error) {
	// Prepare the command to execute with the provided arguments.
	input.Spec.Action = w.path // Set the action to the path.

	// Rewrite the args to be a single string.
	input.Spec.Args = []string{strings.Join(input.Spec.Args, " ")}

	runner, err := NewCmdRunner()
	if err != nil {
		return dry.Wrapf2((*domain.RunnerResult)(nil), err, "failed to create command runner")
	}
	res, err := runner.Execute(ctx, input)
	return dry.Wrapf2(res, err, "failed to execute command")
}
