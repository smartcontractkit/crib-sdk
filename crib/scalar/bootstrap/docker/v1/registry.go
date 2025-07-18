package dockerv1

import (
	"context"
	"embed"
	"os"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/adapter/filehandler"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"

	clientsideapply "github.com/smartcontractkit/crib-sdk/crib/scalar/clientsideapply/v1"
)

const (
	scriptFile = "registry.sh"
)

//go:embed registry.sh
var f embed.FS

var scriptPath string

type Props struct {
	Name string `validate:"required"`
	Port string `validate:"required"`
}

func (p *Props) Validate(ctx context.Context) error {
	return internal.ValidatorFromContext(ctx).Struct(p)
}

// Component returns a new docker registry component. This utilizes a client-side apply
// to create a local docker registry container with the specified name and port.
func Component(name, port string) crib.ComponentFunc {
	return func(ctx context.Context) (crib.Component, error) {
		props := &Props{Name: name, Port: port}
		if err := props.Validate(ctx); err != nil {
			return nil, err
		}
		return dockerRegistry(ctx, props)
	}
}

func dockerRegistry(ctx context.Context, props crib.Props) (crib.Component, error) {
	registryProps := dry.MustAs[*Props](props)
	c, err := clientsideapply.New(ctx, &clientsideapply.Props{
		Namespace: domain.DefaultNamespace,
		OnFailure: domain.FailureAbort,
		Action:    domain.ActionCmd,
		Args: []string{
			"REGISTRY_NAME=" + registryProps.Name,
			"REGISTRY_PORT=" + registryProps.Port,
			scriptPath,
		},
	})
	return dry.Wrap2(c, err)
}

func init() {
	ctx := context.Background()
	eh, err := filehandler.NewReadOnlyFS(f)
	if err != nil {
		panic("failed to create file handler for registry script: " + err.Error())
	}
	fh, err := filehandler.NewTempHandler(ctx, "registry")
	if err != nil {
		panic("failed to create file handler for registry script: " + err.Error())
	}
	if err := filehandler.CopyDir(eh, fh); err != nil {
		panic("failed to copy registry script files: " + err.Error())
	}
	scriptPath = fh.AbsPathFor(scriptFile)
	// Make the script executable
	if err := os.Chmod(scriptPath, 0o755); err != nil { //nolint:gosec // G302
		panic("failed to make registry script executable: " + err.Error())
	}
}
