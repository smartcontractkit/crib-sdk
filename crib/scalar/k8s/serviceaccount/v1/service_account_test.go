package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
)

func TestServiceAccount_New(t *testing.T) {
	t.Parallel()
	internal.JSIIKernelMutex.Lock()
	defer internal.JSIIKernelMutex.Unlock()

	app := internal.NewTestApp(t)
	ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

	t.Run("creates service account with valid props", func(t *testing.T) {
		is := assert.New(t)
		must := require.New(t)

		props := &Props{
			Namespace:   "test-namespace",
			AppName:     "test-app",
			AppInstance: "test-instance",
			Name:        "test-service-account",
		}

		sa, err := New(ctx, props)
		must.NoError(err)
		is.NotNil(sa)
		is.Equal("test-service-account", sa.Name())
		internal.SynthAndSnapYamls(t, app)
	})

	t.Run("creates service account with all optional fields", func(t *testing.T) {
		is := assert.New(t)
		must := require.New(t)

		props := &Props{
			Namespace:                    "test-namespace",
			AppName:                      "test-app",
			AppInstance:                  "test-instance",
			Name:                         "full-service-account",
			Secrets:                      []string{"secret1", "secret2"},
			ImagePullSecrets:             []string{"pull-secret1"},
			AutomountServiceAccountToken: dry.ToPtr(false),
		}

		sa, err := New(ctx, props)
		must.NoError(err)
		is.NotNil(sa)
		is.Equal("full-service-account", sa.Name())
		is.Equal("test-namespace", sa.GetNamespace())
		internal.SynthAndSnapYamls(t, app)
	})

	t.Run("fails with missing required fields", func(t *testing.T) {
		is := assert.New(t)

		props := &Props{
			Namespace: "test-namespace",
			// Missing required fields
		}

		sa, err := New(ctx, props)
		is.Error(err)
		is.Nil(sa)
	})
}

func TestProps_Validate(t *testing.T) {
	t.Parallel()
	internal.JSIIKernelMutex.Lock()
	defer internal.JSIIKernelMutex.Unlock()

	app := internal.NewTestApp(t)
	ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

	t.Run("validates correct props", func(t *testing.T) {
		is := assert.New(t)

		props := &Props{
			Namespace:   "test-namespace",
			AppName:     "test-app",
			AppInstance: "test-instance",
			Name:        "test-service-account",
		}

		err := props.Validate(ctx)
		is.NoError(err)
	})

	t.Run("fails validation with missing required fields", func(t *testing.T) {
		is := assert.New(t)

		props := &Props{
			Namespace: "test-namespace",
			// Missing required fields
		}

		err := props.Validate(ctx)
		is.Error(err)
	})
}

func TestServiceAccount_Methods(t *testing.T) {
	t.Parallel()
	internal.JSIIKernelMutex.Lock()
	defer internal.JSIIKernelMutex.Unlock()

	is := assert.New(t)

	sa := &ServiceAccount{
		props: Props{
			Namespace: "test-namespace",
			Name:      "test-service-account",
		},
	}

	is.Equal("test-service-account", sa.Name())
	is.Equal("test-service-account", sa.GetServiceAccountName())
	is.Equal("test-namespace", sa.GetNamespace())
}
