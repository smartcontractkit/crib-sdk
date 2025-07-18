package helm

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"go/format"
	"path/filepath"
	"strings"
	"text/template" // Explicitly using text/template as we are writing go code.

	"github.com/Masterminds/sprig/v3"
	"github.com/sourcegraph/conc/pool"
	"gopkg.in/yaml.v3"

	"github.com/smartcontractkit/crib-sdk/internal/adapter/filehandler"
	"github.com/smartcontractkit/crib-sdk/internal/adapter/mempools"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
	"github.com/smartcontractkit/crib-sdk/internal/core/port"
)

//go:embed templates/*
var templates embed.FS

var tpl *template.Template

// Generator is a Go Template generator for Helm charts.
type Generator struct {
	fh           port.FileWriter
	TemplateOpts TemplateOpts
}

// TemplateOpts is the struct passed into the Go templates for rendering Helm charts.
type TemplateOpts struct {
	*Defaults
	PackageName  string
	DefaultsFile string
}

// NewGenerator initializes a new Generator instance capable of templating a Helm Chart Scalar Component
// with the provided context, defaults, and output directory.
func NewGenerator(ctx context.Context, d *Defaults, outdir string) (*Generator, error) {
	// Create the file handler for writing files.
	fh, err := filehandler.New(ctx, outdir)
	if err != nil {
		return nil, fmt.Errorf("creating file handler: %w", err)
	}

	// Initialize the generator with default values.
	return &Generator{
		fh: fh,
		TemplateOpts: TemplateOpts{
			DefaultsFile: domain.HelmDefaultsFileName,
			Defaults:     d,
		},
	}, nil
}

func (g *Generator) normalizePackageName() string {
	name := g.TemplateOpts.Release.ReleaseName
	// Trim leading and trailing whitespace.
	name = strings.TrimSpace(name)
	// lower case the string.
	name = strings.ToLower(name)
	// Remove dashes, spaces, and underscores.
	r := strings.NewReplacer("-", "", " ", "", "_", "")
	name = r.Replace(name)
	// Convert to utf-8.
	name = strings.ToValidUTF8(name, "_") // Replace invalid UTF-8 characters with underscores.
	// Replace any non-alphanumeric characters with underscores.
	name = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' {
			return r // Keep alphanumeric and underscore characters.
		}
		return '_' // Replace other characters with underscore.
	}, name)
	// Ensure the name starts with a letter.
	if name != "" && (name[0] < 'a' || name[0] > 'z') {
		name = "pkg_" + name // Prefix with "pkg_" if it doesn't start with a letter.
	}
	return name
}

func (g *Generator) Generate(ctx context.Context) error {
	g.TemplateOpts.PackageName = g.normalizePackageName()
	return dry.FirstErrorFns(
		func() error { return g.fh.MkdirAll("testdata", 0o755) },
		func() error { return g.Render() },
		func() error { return g.TemplateOpts.Save(ctx, g.fh) },
		func() error { return g.CopyValues() },
	)
}

func (g *Generator) CopyValues() (err error) {
	// Create the values file in testdata/values.yaml.
	f, err := g.fh.Create(filepath.Join("testdata", domain.HelmValuesFileName))
	if err != nil {
		return fmt.Errorf("creating values file: %w", err)
	}
	defer func() {
		err = errors.Join(err, f.Close())
	}()

	// Marshal the values from the Defaults struct into YAML format.
	return yaml.NewEncoder(f).Encode(g.TemplateOpts.Values)
}

func (g *Generator) Render() error {
	// Ensure the package name is set.
	if g.TemplateOpts.Release.ReleaseName == "" {
		return errors.New("release name is required for template rendering")
	}

	// Render the templates.
	wg := pool.New().WithErrors()
	for _, t := range tpl.Templates() {
		if !strings.HasSuffix(t.Name(), ".gotemplate") {
			continue // Skip non-template files.
		}
		wg.Go(func() error { return g.render(t.Name()) })
	}
	return wg.Wait()
}

// render renders the named template to file.
func (g *Generator) render(name string) (err error) {
	dst := strings.ReplaceAll(name, ".gotemplate", ".go")

	// Execute the template with the provided Generator data.
	buf, ret := mempools.BytesBuffer.Get()
	defer ret()
	if err := tpl.ExecuteTemplate(buf, name, g.TemplateOpts); err != nil {
		return fmt.Errorf("rendering template: %w", err)
	}
	src, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("formatting generated code: %w", err)
	}

	f, err := g.fh.Create(dst)
	if err != nil {
		return fmt.Errorf("creating file %q: %w", dst, err)
	}
	defer func() {
		err = errors.Join(err, f.Close())
	}()

	_, err = f.Write(src)
	return dry.Wrapf(err, "writing to file %q", dst)
}

func init() {
	tpl = template.Must(
		template.New("helm").
			Funcs(sprig.FuncMap()).
			ParseFS(templates, "templates/*.gotemplate"),
	)
}
