package clientsideapply

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/smartcontractkit/crib-sdk/internal/adapter/mempools"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
)

func (e *EchoRunner) Execute(ctx context.Context, input *domain.ClientSideApplyManifest) (*domain.RunnerResult, error) {
	buf, reset := mempools.BytesBuffer.Get()
	defer reset()

	buf.WriteString("ClientSideApply:\n")
	fmt.Fprintf(buf, "  OnFailure: %s\n", input.Spec.OnFailure)
	fmt.Fprintf(buf, "  Action: %s\n", input.Spec.Action)
	buf.WriteString("  Args:\n")
	for _, arg := range input.Spec.Args {
		fmt.Fprintf(buf, "    - %s\n", arg)
	}
	_, err := io.Copy(e.w, bytes.NewReader(buf.Bytes()))
	return &domain.RunnerResult{
		Output: buf.Bytes(),
	}, err
}
