package clientsideapply

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
	"github.com/smartcontractkit/crib-sdk/internal/core/port"
)

func TestImplements(t *testing.T) {
	t.Parallel()

	is := assert.New(t)

	is.Implements((*port.ClientSideApplyRunner)(nil), new(EchoRunner))
	is.Implements((*port.ClientSideApplyRunner)(nil), new(CmdRunner))
	is.Implements((*port.ClientSideApplyRunner)(nil), new(wrappedRunner))
}

func TestNewEchoRunner(t *testing.T) {
	t.Parallel()

	is := assert.New(t)

	runner, err := NewEchoRunner(nil)
	is.NoError(err)
	is.NotNil(runner)
}

func TestNewCmdRunner(t *testing.T) {
	t.Parallel()

	is := assert.New(t)

	runner, err := NewCmdRunner()
	is.NoError(err)
	is.NotNil(runner)
}

func TestNewCribctlRunner(t *testing.T) {
	t.Parallel()

	is := assert.New(t)

	runner, err := NewCribctlRunner()
	if errors.Is(err, domain.ErrNotFoundInPath) {
		t.Skip("cribctl not available to testing framework.")
	}
	is.NoError(err)
	is.NotNil(runner)
}

func TestNewKubectlRunner(t *testing.T) {
	t.Parallel()

	is := assert.New(t)

	runner, err := NewKubectlRunner()
	if errors.Is(err, domain.ErrNotFoundInPath) {
		t.Skip("kubectl not available to testing framework.")
	}
	is.NoError(err)
	is.NotNil(runner)
}

func TestNewRunner(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		manifest *domain.ClientSideApplyManifest
		wantRes  []byte
	}{
		{
			name: "CmdRunner",
			manifest: &domain.ClientSideApplyManifest{
				Spec: domain.ClientSideApplySpec{
					OnFailure: "abort",
					Action:    "cmd",
					Args: []string{
						"echo",
						"Hello, World!",
					},
				},
			},
			wantRes: []byte("Hello, World!\n"),
		},
		{
			name: "KubectlRunner",
			manifest: &domain.ClientSideApplyManifest{
				Spec: domain.ClientSideApplySpec{
					OnFailure: "abort",
					Action:    "kubectl",
					Args: []string{
						"apply",
						"-f",
						"example.yaml",
					},
				},
			},
			wantRes: []byte("apply -f example.yaml\n"),
		},
		{
			name: "CribctlRunner",
			manifest: &domain.ClientSideApplyManifest{
				Spec: domain.ClientSideApplySpec{
					OnFailure: "abort",
					Action:    "cribctl",
					Args: []string{
						"apply",
						"plan",
						"example",
					},
				},
			},
			wantRes: []byte("apply plan example\n"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			runner, err := NewRunner(tc.manifest)
			require.NoError(t, err)
			require.NotNil(t, runner)

			result, err := runner.Execute(t.Context(), tc.manifest)
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Contains(t, string(result.Output), string(tc.wantRes), "Expected output to match the expected result")
		})
	}
}

func TestKubectlExecute(t *testing.T) {
	t.Parallel()

	input := &domain.ClientSideApplyManifest{
		Spec: domain.ClientSideApplySpec{
			OnFailure: "abort",
			Action:    "kubectl",
			Args: []string{
				"apply",
				"-f",
				"example.yaml",
			},
		},
	}

	runner, err := NewKubectlRunner()
	require.NoError(t, err)

	result, err := runner.Execute(t.Context(), input)
	require.NoError(t, err)
	require.NotEmpty(t, result.Output, "Expected non-empty output from kubectl execution")

	resStr := string(result.Output)
	assert.Contains(t, resStr, "apply -f example.yaml", "Expected output to contain 'apply -f example.yaml'")
}

func TestCribctlExecute(t *testing.T) {
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

	runner, err := NewCribctlRunner()
	require.NoError(t, err)

	result, err := runner.Execute(t.Context(), input)
	require.NoError(t, err)
	require.NotEmpty(t, result.Output, "Expected non-empty output from kubectl execution")

	resStr := string(result.Output)
	assert.Contains(t, resStr, "apply plan example", "Expected output to contain 'apply plan example'")
}
