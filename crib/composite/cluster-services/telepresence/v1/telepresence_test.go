package telepresencev1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/crib-sdk/internal"
)

func TestTelepresenceComponent(t *testing.T) {
	t.Parallel()
	internal.JSIIKernelMutex.Lock()
	t.Cleanup(internal.JSIIKernelMutex.Unlock)

	t.Run("valid props", func(t *testing.T) {
		app := internal.NewTestApp(t)
		ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

		props := &Props{
			Namespace: "test-namespace",
		}

		component := Component(props)
		require.NotNil(t, component)

		result, err := component(ctx)
		require.NoError(t, err)
		assert.NotNil(t, result)

		internal.SynthAndSnapYamls(t, app)
	})

	t.Run("invalid props", func(t *testing.T) {
		app := internal.NewTestApp(t)
		ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

		props := &Props{
			Namespace: "", // empty namespace should fail validation
		}

		component := Component(props)
		require.NotNil(t, component)

		result, err := component(ctx)
		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestProps_Validate(t *testing.T) {
	t.Parallel()
	internal.JSIIKernelMutex.Lock()
	t.Cleanup(internal.JSIIKernelMutex.Unlock)

	t.Run("valid props", func(t *testing.T) {
		app := internal.NewTestApp(t)
		ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

		props := &Props{
			Namespace: "test-namespace",
		}

		err := props.Validate(ctx)
		assert.NoError(t, err)
	})

	t.Run("empty namespace", func(t *testing.T) {
		app := internal.NewTestApp(t)
		ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

		props := &Props{
			Namespace: "",
		}

		err := props.Validate(ctx)
		assert.Error(t, err)
	})
}
