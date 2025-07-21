// Package clientsideapply provides adapters for client-side apply operations.
// Each sub-adapter implements [port.ClientSideApplyRunner].
package clientsideapply

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"testing"

	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
	"github.com/smartcontractkit/crib-sdk/internal/core/port"
)

type (
	// EchoRunner is a client-side apply runner that writes arguments to a writer.
	EchoRunner struct {
		w io.Writer
	}

	// CmdRunner is a client-side apply runner that executes commands using the command line.
	CmdRunner struct{}

	// wrappedRunner is a client-side apply runner that wraps another runner and executes a command.
	wrappedRunner struct {
		// Path to the binary to execute.
		path string
	}
)

// NewRunner creates a new ClientSideApplyRunner based on the manifest's action.
func NewRunner(manifest *domain.ClientSideApplyManifest) (port.ClientSideApplyRunner, error) {
	if manifest == nil {
		return nil, errors.New("manifest cannot be nil")
	}

	switch manifest.Spec.Action {
	case domain.ActionCmd:
		return NewCmdRunner()
	case domain.ActionKubectl:
		return NewKubectlRunner()
	case domain.ActionCribctl:
		return NewCribctlRunner()
	case domain.ActionTask:
		return NewTaskRunner()
	default:
		return newWrappedRunner(manifest.Spec.Action)
	}
}

// NewEchoRunner creates a new EchoRunner with the given writer.
func NewEchoRunner(w io.Writer) (port.ClientSideApplyRunner, error) {
	return &EchoRunner{w: w}, nil
}

// NewCmdRunner creates a new CmdRunner.
func NewCmdRunner() (port.ClientSideApplyRunner, error) {
	return &CmdRunner{}, nil
}

// NewCribctlRunner creates a new CribctlRunner with the given path.
func NewCribctlRunner() (port.ClientSideApplyRunner, error) {
	return newWrappedRunner(domain.ActionCribctl)
}

// NewKubectlRunner creates a new KubectlRunner with the given path.
func NewKubectlRunner() (port.ClientSideApplyRunner, error) {
	return newWrappedRunner(domain.ActionKubectl)
}

// NewTaskRunner creates a new TaskRunner with the given path.
func NewTaskRunner() (port.ClientSideApplyRunner, error) {
	return newWrappedRunner(domain.ActionTask)
}

// NewHelmRunner creates a new HelmRunner with the given path.
func NewHelmRunner() (port.ClientSideApplyRunner, error) {
	return newWrappedRunner(domain.ActionHelm)
}

func newWrappedRunner(executable string) (port.ClientSideApplyRunner, error) {
	// If we're running under test, prefix the binary with "echo " to avoid executing it.
	if testing.Testing() && os.Getenv("CRIB_ENABLE_COMMAND_EXECUTION") == "" {
		return &wrappedRunner{path: "echo " + executable}, nil
	}

	path, err := exec.LookPath(executable)
	if err != nil {
		return nil, domain.NewNotFoundInPathError(executable)
	}
	return &wrappedRunner{path: path}, nil
}
