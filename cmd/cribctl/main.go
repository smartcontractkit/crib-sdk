package main

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/smartcontractkit/crib-sdk/cmd/cribctl/cmd"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// This allows children to add more hooks without overriding the parent.
	// The default behavior would reset the hook.
	cobra.EnableTraverseRunHooks = true

	cmd.Execute(ctx)
}
