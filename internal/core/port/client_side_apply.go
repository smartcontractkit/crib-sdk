package port

import (
	"context"

	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
)

// ClientSideApplyRunner defines the interface for executing client-side apply operations.
type ClientSideApplyRunner interface {
	// Execute executes the client-side apply manifest and returns the result.
	Execute(ctx context.Context, input *domain.ClientSideApplyManifest) (*domain.RunnerResult, error)
}
