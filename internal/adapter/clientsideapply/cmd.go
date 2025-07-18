package clientsideapply

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/samber/lo"
	"github.com/theckman/yacspin"

	"github.com/smartcontractkit/crib-sdk/internal/adapter/mempools"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
)

// Provide a global mutex to ensure that only one command is executed at a time.
// This is because we overwrite the stdin/stdout/stderr of the process while the command is running.
var mu sync.Mutex

func (c *CmdRunner) Execute(ctx context.Context, input *domain.ClientSideApplyManifest) (*domain.RunnerResult, error) {
	const cmd = "/bin/bash"
	action := input.Spec.Action
	if action == "" {
		return nil, domain.ErrEmptyAction
	}
	if action == domain.ActionCmd {
		action = "" // Unset the action to use the default shell command
	}

	// Prepend the action to the command arguments.
	input.Spec.Args = lo.Compact(append([]string{action}, input.Spec.Args...))
	args := []string{"-c", strings.Join(input.Spec.Args, " ")}

	e := exec.CommandContext(ctx, cmd, args...) //nolint:gosec // Needed for command execution.
	// Copy the environment variables from the current process.
	e.Env = os.Environ()

	// Capture the current process's stdin, stdout, and stderr.
	// And scaffold a restore mechanism.
	// This is necessary because we will overwrite these with our own pipe.
	mu.Lock()
	defer mu.Unlock()

	spin, stop := showProgress()
	spin.StopMessage(fmt.Sprintf("Executing %s", input.Spec.Action))

	// Run the command, collecting the output of the command.
	// Note: The output should have also been streamed to stdout/stderr.
	// Possible gotcha here, we may need to inspect the output of the command
	// to fully determine success or failure and not just the exit code.
	res, err := combinedOutput(e)
	if err != nil {
		_ = spin.StopFail()
		return nil, err
	}
	stop()

	return &domain.RunnerResult{
		Output: res,
	}, nil
}

// combinedOutput runs cmd, writing its stdout to [os.Stdout] and
// stderr to [os.Stderr], while also capturing both into one [*bytes.Buffer].
func combinedOutput(cmd *exec.Cmd) ([]byte, error) {
	buf, reset := mempools.BytesBuffer.Get()
	defer reset()
	// tee stdout to both os.Stdout and buf
	cmd.Stdout = buf
	// tee stderr to both os.Stderr and buf
	cmd.Stderr = buf
	err := cmd.Run()
	return dry.Wrapf2(buf.Bytes(), err, "running command %q", strings.Join(cmd.Args, " "))
}

func showProgress() (spinner *yacspin.Spinner, stopFn func()) {
	spinner, err := yacspin.New(spinnerCfg)
	if err != nil {
		return spinner, func() {}
	}
	if err := spinner.Start(); err != nil {
		return spinner, func() {}
	}
	return spinner, func() {
		_ = spinner.Stop()
	}
}
