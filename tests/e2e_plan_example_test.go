package tests

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal/adapter/clientsideapply"
	"github.com/smartcontractkit/crib-sdk/internal/adapter/filehandler"
	"github.com/smartcontractkit/crib-sdk/internal/adapter/mempools"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
	"github.com/smartcontractkit/crib-sdk/internal/core/service"

	nginxcontroller "github.com/smartcontractkit/crib-sdk/crib/composite/cluster-services/nginx-controller/v1"
	kind "github.com/smartcontractkit/crib-sdk/crib/scalar/bootstrap/kind/v1"
	clientsideapplyv1 "github.com/smartcontractkit/crib-sdk/crib/scalar/clientsideapply/v1"
)

type E2ESuite struct {
	suite.Suite

	plan *crib.Plan
	fh   *filehandler.Handler
}

func TestE2ESuite(t *testing.T) {
	// Skip running under CI. CI has a different version of Kind and it does fun things.
	if os.Getenv("CI") != "" {
		t.Skip("Skipping e2e test in CI environment")
	}
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}
	for _, dep := range []string{domain.ActionKind, domain.ActionKubectl} {
		if _, err := exec.LookPath(dep); err != nil {
			t.Skipf("⚠️  Skipping e2e test because %q is not available in PATH", dep)
		}
	}
	// Set environment variable to enable client-side apply.
	t.Setenv("CRIB_ENABLE_COMMAND_EXECUTION", "true")
	suite.Run(t, new(E2ESuite))
}

func (s *E2ESuite) SetupSuite() {
	cleanup(s.T())
}

func (s *E2ESuite) TearDownSuite() {
	cleanup(s.T())
}

func (s *E2ESuite) SetupTest() {
	cleanup(s.T())
	ctx := s.T().Context()

	u := uuid.NewString()[:8] // Use the first 8 characters of the UUID for brevity.
	cluster := fmt.Sprintf("example-%s", u)

	runner, err := clientsideapply.NewCmdRunner()
	s.Require().NoError(err, "failed to create client-side apply runner")
	assert.Eventually(s.T(), func() bool {
		clusters, _ := runner.Execute(ctx, &domain.ClientSideApplyManifest{
			Spec: domain.ClientSideApplySpec{
				Action: domain.ActionKind,
				Args:   []string{"get", "clusters"},
			},
		})
		return !bytes.Contains(clusters.Output, []byte("example-"))
	}, time.Minute, 500*time.Millisecond, "Kind cluster %s should be created", cluster)

	s.plan = crib.NewPlan(
		"E2E Test Plan",
		crib.Namespace(domain.DefaultNamespace),
		crib.ComponentSet(
			// Wait 10 seconds for the cluster to be ready.
			clientsideapplyv1.Component(&clientsideapplyv1.Props{
				Action:    domain.ActionCmd,
				OnFailure: domain.FailureAbort,
				Args:      []string{"sleep", "10"},
			}),
			kind.Component(cluster),
			nginxcontroller.Component(),
		),
	)

	fh, err := filehandler.New(ctx, s.T().TempDir())
	s.Require().NoError(err)
	s.fh = fh
}

func (s *E2ESuite) TestRealWorld() {
	ctx := s.T().Context()
	plan := s.plan
	res, err := plan.Apply(ctx)
	if errors.Is(err, domain.ErrNotFoundInPath) {
		s.T().Skip("Skipping test because required executables are not available in the PATH")
		return
	}
	s.Require().NoError(err, "failed to apply plan")

	ids := slices.Collect(res.ComponentIDs())
	components := slices.Collect(res.Components())
	applies := slices.Collect(res.ComponentByName("sdk.ClientSideApply"))
	controllers := slices.Collect(res.ComponentByName("sdk.NginxController"))

	s.Len(ids, 3, "expected 3 components to be applied, got %d", len(ids))
	s.Len(components, 3, "expected 3 components, got %d", len(components))
	s.Len(applies, 2, "expected 2 ClientSideApply components, got %d", len(applies))
	s.Len(controllers, 1, "expected 1 NginxController component, got %d", len(controllers))

	var args []string
	for component := range res.ComponentByName("sdk.ClientSideApply") {
		csa := crib.ComponentState[*clientsideapplyv1.Result](component)
		args = append(args, strings.Join(csa.Args, " "))
	}
	s.Require().Len(args, 2, "expected 2 sets of arguments for ClientSideApply components, got %d", len(args))
	s.Contains(args[0], "sleep 10", "first ClientSideApply should have 'sleep 10' args")
	s.Contains(args[1], "KIND_CLUSTER_NAME=example-", "second ClientSideApply should have 'create cluster --name example-' args")
}

func (s *E2ESuite) TestPlanExample() {
	ctx := s.T().Context()

	svc, err := service.NewPlanService(ctx, s.fh)
	s.Require().NoError(err, "failed to create plan service")
	s.Require().NotNil(svc, "plan service should not be nil")

	plan, err := svc.CreatePlan(ctx, s.plan)
	s.Require().NoError(err, "failed to create plan")
	s.Require().NotNil(plan, "plan should not be nil")

	_, err = plan.Apply(ctx)
	if errors.Is(err, domain.ErrNotFoundInPath) {
		s.T().Skip("Skipping test because required executables are not available in the PATH")
		return
	}
	s.Require().NoError(err, "failed to apply plan")
}

func cleanup(t *testing.T) {
	t.Helper()
	must := require.New(t)

	ctx := context.WithoutCancel(t.Context())

	runner, err := clientsideapply.NewCmdRunner()
	must.NoError(err, "failed to create client-side apply runner")

	// Use the runner to get clusters.
	clusters, err := runner.Execute(ctx, &domain.ClientSideApplyManifest{
		Spec: domain.ClientSideApplySpec{
			Action: domain.ActionKind,
			Args:   []string{"get", "clusters"},
		},
	})
	must.NoError(err, "failed to get clusters")
	must.NotNil(clusters, "clusters should not be nil")

	buf, reset := mempools.BytesBuffer.Get()
	t.Cleanup(reset)
	buf.Write(clusters.Output)
	scanner := bufio.NewScanner(buf)

	// Loop over each cluster and print its name.
	var cleanupClusters []struct {
		name string
		*domain.ClientSideApplyManifest
	}
	for scanner.Scan() {
		if !strings.HasPrefix(scanner.Text(), "example-") {
			continue // Skip clusters that do not start with "example-"
		}
		cleanupClusters = append(cleanupClusters, struct {
			name string
			*domain.ClientSideApplyManifest
		}{
			name: scanner.Text(),
			ClientSideApplyManifest: &domain.ClientSideApplyManifest{
				Spec: domain.ClientSideApplySpec{
					Action: domain.ActionKind,
					Args:   []string{"delete", "cluster", "-n", scanner.Text()},
				},
			},
		})
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		must.NoError(err, "error reading clusters output")
	}

	var cleanupErrs error
	for _, cleanup := range cleanupClusters {
		t.Logf("Cleaning up cluster: %s", cleanup.name)
		_, err := runner.Execute(ctx, cleanup.ClientSideApplyManifest)
		cleanupErrs = errors.Join(cleanupErrs, err)
	}
	assert.NoError(t, cleanupErrs, "failed to clean up clusters")
}
