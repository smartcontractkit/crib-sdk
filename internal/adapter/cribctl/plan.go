package cribctl

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/smartcontractkit/crib-sdk/contrib"
	"github.com/smartcontractkit/crib-sdk/internal/adapter/filehandler"
	"github.com/smartcontractkit/crib-sdk/internal/core/service"
)

// ValidatePlanArgs validates the arguments for plan-related commands.
// It ensures exactly one argument is provided and that the plan exists.
func ValidatePlanArgs(command string) func(*cobra.Command, []string) error {
	const errMsg = "command requires exactly one argument: the name of the plan"
	return func(cmd *cobra.Command, args []string) (err error) {
		availablePlans := contrib.Plans()
		msg := fmt.Sprintf("\n\nAvailable plans:\n- %s\n\n", strings.Join(availablePlans, "\n- "))

		if err := cobra.ExactArgs(1)(cmd, args); err != nil {
			return fmt.Errorf("%s: %s. %s", command, errMsg, msg)
		}
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("%s: no plan found with name %q. %s", command, args[0], msg)
			}
		}()
		_ = contrib.Plan(args[0])
		return nil
	}
}

// PreviewPlan previews a CRIB-SDK Plan by its name and returns the DAG as a tree.
// If outputDir is provided, the generated files will be dumped to that directory.
// If outputDir is empty, a temporary directory will be used.
func PreviewPlan(ctx context.Context, fh *filehandler.Handler, name string) (preview, outputPath string, err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	plan := contrib.Plan(name)
	if plan == nil {
		return "", "", fmt.Errorf("no plan found with name %s", name)
	}

	// Create a new PlanService with a temporary directory.
	svc, err := service.NewPlanService(ctx, fh)
	if err != nil {
		return "", "", fmt.Errorf("failed to create plan service: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Resolving plan dependencies for plan %q.\n", name)
	appPlan, err := svc.CreatePlan(ctx, plan)
	if err != nil {
		return "", "", fmt.Errorf("failed to create plan: %w", err)
	}

	// Return the preview and the output directory path
	return appPlan.Preview(ctx), fh.Name(), nil
}

// ApplyPlan applies a CRIB-SDK Plan by its name.
func ApplyPlan(ctx context.Context, fh *filehandler.Handler, name string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	plan := contrib.Plan(name)
	if plan == nil {
		return fmt.Errorf("no plan found with name %s", name)
	}
	// Create a new PlanService.
	svc, err := service.NewPlanService(ctx, fh)
	if err != nil {
		return fmt.Errorf("failed to create plan service: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Resolving plan dependencies for plan %q.\n", name)
	appPlan, err := svc.CreatePlan(ctx, plan)
	if err != nil {
		return fmt.Errorf("failed to create plan: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Applying plan %q.\n", name)
	_, err = appPlan.Apply(ctx)
	return err
}
