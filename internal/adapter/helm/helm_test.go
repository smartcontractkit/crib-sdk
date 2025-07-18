package helm

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/alexflint/go-arg"
	"github.com/gkampitakis/go-snaps/match"
	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/smartcontractkit/crib-sdk/internal/adapter/clientsideapply"
	"github.com/smartcontractkit/crib-sdk/internal/adapter/filehandler"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
	"github.com/smartcontractkit/crib-sdk/internal/core/port"
)

func TestISatisfies(t *testing.T) {
	t.Parallel()
	is := assert.New(t)

	is.Implements((*port.HelmClient)(nil), &Client{})
}

type (
	TemplateCmd struct {
		Chart  string   `arg:"positional"`
		Folder string   `arg:"positional"`
		Values []string `arg:"-f,separate"`
	}

	CmdArgs struct {
		Template *TemplateCmd `arg:"subcommand:template"`
	}
)

func parseResponse(t *testing.T, data []byte) *CmdArgs {
	t.Helper()
	must := require.New(t)

	must.NotNil(data)

	cmdFields := strings.Fields(string(data))
	must.GreaterOrEqual(len(cmdFields), 2, "command must have at least two fields: command name and subcommand")
	must.Equal("helm", cmdFields[0], "first field must be 'helm'")

	var args CmdArgs
	p, err := arg.NewParser(arg.Config{
		StrictSubcommands: true,
	}, &args)
	must.NoError(err)
	must.NoError(p.Parse(cmdFields[1:]))
	return &args
}

type HelmSuite struct {
	suite.Suite
	fh      port.FileHandler
	client  port.HelmClient
	release port.ChartReleaser
	snaps   *snaps.Config

	mu sync.Mutex
}

func TestHelmSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(HelmSuite))
}

func (s *HelmSuite) SetupSuite() {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := s.T().Context()
	localFs, err := filehandler.New(ctx, "testdata")
	s.Require().NoError(err)

	tmpFs, err := filehandler.NewTempHandler(ctx, s.T().Name())
	s.Require().NoError(err)
	s.fh = tmpFs

	// Copy values.yaml to the temporary filesystem.
	s.Require().NoError(filehandler.CopyFile(localFs, tmpFs, "values.yaml"))

	// Copy Chart1.yaml to Chart.yaml in the temporary filesystem.
	testChart, err := localFs.Open("Chart1.yaml")
	s.Require().NoError(err)

	chart, err := tmpFs.Create("Chart.yaml")
	s.Require().NoError(err)

	_, err = io.Copy(chart, testChart)
	s.Require().NoError(err)
	s.Require().NoError(chart.Close())
	s.Require().NoError(testChart.Close())
}

func (s *HelmSuite) SetupTest() {
	s.mu.Lock()
	defer s.mu.Unlock()

	c, err := NewClient(s.T().Context())
	s.Require().NoError(err)
	s.client = c

	s.release = &Release{
		Name:        "test-chart",
		ReleaseName: "test-release",
		Repository:  "https://example.com/charts",
		Version:     "1.0.0",
	}
}

func (s *HelmSuite) BeforeTest(suiteName, testName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.snaps = snaps.WithConfig(
		snaps.Filename(suiteName + "_" + testName),
	)
}

func (s *HelmSuite) TestVendorRepo() {
	fh, err := s.client.VendorRepo(s.T().Context(), s.release)
	s.Require().NoError(err)
	s.NotNil(fh, "expected a non-nil file handler")
	s.DirExists(fh.Name(), "expected the chart directory to exist")
}

func (s *HelmSuite) TestAddRepo() {
	s.NoError(s.client.AddRepo(s.T().Context(), s.release))
}

func (s *HelmSuite) TestUpdateRepo() {
	s.NoError(s.client.UpdateRepo(s.T().Context(), s.release))
}

func (s *HelmSuite) TestPullRepo() {
	t := s.T()
	must := s.Require()
	fh, err := s.client.PullRepo(t.Context(), s.release)
	must.NoError(err)
	must.NotNil(fh)
	must.DirExists(fh.Name())

	s.Equal(dry.As[*Release](s.release).Name, filepath.Base(fh.Name()), "expected the chart directory name to match the release name")
}

func (s *HelmSuite) TestTemplateRepo() {
	t := s.T()
	must := s.Require()
	got, err := s.client.TemplateRepo(t.Context(), s.release, s.fh)
	must.NoError(err)

	cmd := parseResponse(t, got)
	must.NotNil(cmd.Template, "expected Template command to be present")

	s.snaps.MatchJSON(t, cmd,
		match.Custom("Template.Chart", func(val any) (any, error) {
			value, ok := val.(string)
			if !ok {
				return nil, fmt.Errorf("expected string but got %T", val)
			}

			if strings.Contains(value, "/") {
				return nil, fmt.Errorf("expected chart name to not contain slashes but got %s", value)
			}

			if want := dry.As[*Release](s.release).ReleaseName; value != want {
				return nil, fmt.Errorf("expected chart name to be %s but got %s", want, value)
			}

			return val, nil
		}),
		match.Custom("Template.Folder", func(val any) (any, error) {
			folder, ok := val.(string)
			if !ok {
				return false, fmt.Errorf("expected string but got %T", val)
			}

			if folder != s.fh.Name() {
				return false, fmt.Errorf("expected folder to be %s but got %s", s.fh.Name(), folder)
			}

			return true, nil
		}),
		match.Custom("Template.Values", func(val any) (any, error) {
			values, ok := val.([]any)
			if !ok {
				return false, fmt.Errorf("expected []any but got %T", val)
			}

			if len(values) != 1 {
				return false, fmt.Errorf("expected exactly one value file but got %d", len(values))
			}

			if got, want := dry.As[string](values[0]), s.fh.AbsPathFor(domain.HelmValuesFileName); got != want {
				return false, fmt.Errorf("expected values file path to be equal, got %s want %s", got, want)
			}

			return true, nil
		}),
	)
}

func Test_runCommand(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	r, err := clientsideapply.NewCmdRunner()
	must.NoError(err)

	c := &Client{executor: r}

	res, err := c.runCommand(t.Context(), "--help")
	must.NoError(err, "expected no error running command")
	assert.Contains(t, string(res.Output), "The Kubernetes package manager")
}
