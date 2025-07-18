package kindv1

import (
	"context"
	"embed"
	"fmt"
	"os"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/adapter/filehandler"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"

	clientsideapply "github.com/smartcontractkit/crib-sdk/crib/scalar/clientsideapply/v1"
)

const (
	defaultsFile = "kind.defaults.yaml"
	scriptFile   = "kind.sh"
)

//go:embed kind.defaults.yaml
//go:embed kind.sh
var f embed.FS

var (
	defaultsPath string
	scriptPath   string
)

type Props struct {
	Name string `validate:"required"`
}

func (p *Props) Validate(ctx context.Context) error {
	return internal.ValidatorFromContext(ctx).Struct(p)
}

// Component returns a new kind cluster component. This utilizes a client-side apply
// to create a kind cluster with the specified name. The cluster is created using the
// `kind create cluster` command with a default configuration file.
func Component(name string) crib.ComponentFunc {
	return func(ctx context.Context) (crib.Component, error) {
		props := &Props{Name: name}
		if err := props.Validate(ctx); err != nil {
			return nil, err
		}
		return kindCluster(ctx, props)
	}
}

func kindCluster(ctx context.Context, props crib.Props) (crib.Component, error) {
	kindProps := dry.MustAs[*Props](props)
	c, err := clientsideapply.New(ctx, &clientsideapply.Props{
		Namespace: domain.DefaultNamespace,
		OnFailure: domain.FailureAbort,
		Action:    domain.ActionCmd,
		Args: []string{
			fmt.Sprintf("KIND_CONFIG_FILE=%s", defaultsPath),
			fmt.Sprintf("KIND_CLUSTER_NAME=%s", kindProps.Name),
			scriptPath,
		},
	})
	return dry.Wrap2(c, err)
}

func init() {
	ctx := context.Background()
	eh, err := filehandler.NewReadOnlyFS(f)
	if err != nil {
		panic("failed to create file handler for kind defaults and script: " + err.Error())
	}
	fh, err := filehandler.NewTempHandler(ctx, "kind")
	if err != nil {
		panic("failed to create file handler for kind defaults: " + err.Error())
	}
	if err := filehandler.CopyDir(eh, fh); err != nil {
		panic("failed to copy kind defaults and script files: " + err.Error())
	}
	scriptPath = fh.AbsPathFor(scriptFile)
	// Make the script executable
	if err := os.Chmod(scriptPath, 0o755); err != nil { //nolint:gosec // G302
		panic("failed to make kind script executable: " + err.Error())
	}
}
