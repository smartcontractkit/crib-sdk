package contrib

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
)

func TestPlan(t *testing.T) {
	t.Parallel()

	const name = "examplev1"
	plan := Plan(name)
	assert.Equal(t, name, plan.Name())
	assert.Equal(t, domain.DefaultNamespace, plan.Namespace())
}

func TestPlans(t *testing.T) {
	t.Parallel()

	assert.Greater(t, len(Plans()), 0)
}
