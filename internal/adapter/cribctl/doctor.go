package cribctl

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"time"

	"github.com/expr-lang/expr"
	"github.com/theckman/yacspin"
	"golang.org/x/mod/semver"

	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/adapter/mempools"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
)

var cribDependencies = []dependency{
	{Name: "kubectl"},
	{
		Name:           "helm",
		VersionMatcher: "'%s'>='3.18.4'",
		VersionCommand: []string{"helm", "version", "--template", "{{.Version}}"},
	},
	{
		Name:           "task",
		VersionMatcher: "'%s'>='3.40.0'",
		VersionCommand: []string{"task", "--version"},
	},
	{
		Name:           "kind",
		VersionMatcher: "'%s'>='0.20.0'",
		VersionCommand: []string{"kind", "--version"},
	},
	{Name: "telepresence"},
	{
		Name:           "node",
		VersionMatcher: "'%s'=='22.17.0'",
		VersionCommand: []string{"node", "-v"},
	},
	{
		Name:           "asdf",
		Optional:       true,
		VersionMatcher: "'%s'>='0.15.0'",
		VersionCommand: []string{"asdf", "--version"},
	},
	{
		Name:           "go",
		VersionMatcher: "'%s'>='1.24.0'",
		VersionCommand: []string{"go", "version"},
	},
}

var (
	spinnerCfg = yacspin.Config{
		Frequency:         100 * time.Millisecond,
		Writer:            os.Stderr,
		CharSet:           yacspin.CharSets[11],
		Suffix:            " ", // puts at least one space between the animating spinner and the Message
		Message:           "Validating cribctl dependencies...",
		SuffixAutoColon:   true,
		ColorAll:          true,
		Colors:            []string{"fgYellow"},
		StopCharacter:     "✓",
		StopColors:        []string{"fgGreen"},
		StopMessage:       "Done!",
		StopFailCharacter: "✗",
		StopFailColors:    []string{"fgRed"},
		StopFailMessage:   "failed",
	}

	errDependencyNotFound = errors.New("required dependency not found")
	versionre             = regexp.MustCompile(`\d+\.\d+(?:\.\d+)?`)
)

type dependency struct {
	Name           string   `validate:"required"`
	VersionMatcher string   `validate:"omitempty,expr"`
	Instructions   string   `validate:"-"`
	VersionCommand []string `validate:"omitempty,required_with=VersionMatcher,dive,required"`
	Optional       bool
}

// DoctorCommand checks for the availability and version of cribctl dependencies.
func DoctorCommand(ctx context.Context) {
	buf, ret := mempools.BytesBuffer.Get()
	defer ret()

	spinner, done := showProgress()
	for _, dep := range cribDependencies {
		localBuf, ret := mempools.BytesBuffer.Get()
		spinner.Message(fmt.Sprintf("Checking for %s...", dep.Name))
		if err := dep.checkAvailability(ctx, localBuf); err == nil && dep.VersionMatcher != "" {
			dep.checkVersion(ctx, localBuf)
		}

		hdr := "%s"
		if dep.Optional {
			hdr = "%s (optional)"
		}
		fmt.Fprintln(buf, fmt.Sprintf(hdr, dep.Name))
		_, _ = io.Copy(buf, localBuf)
		ret()
	}
	done()
	_, _ = io.Copy(os.Stderr, buf)
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

func (d *dependency) checkAvailability(_ context.Context, buf *bytes.Buffer) error {
	bin, err := exec.LookPath(d.Name)
	if err == nil {
		fmt.Fprintf(buf, "  ✅ Found at %s\n", bin)
		return nil
	}

	if d.Optional {
		_, _ = buf.WriteString("  ⚠️  Not found.\n")
		return errDependencyNotFound
	}
	_, _ = buf.WriteString("  ❌ Not found!\n")
	return errDependencyNotFound
}

func (d *dependency) checkVersion(ctx context.Context, buf *bytes.Buffer) {
	if len(d.VersionCommand) == 0 {
		return // No version command to check.
	}

	//nolint:gosec // G204: No user input is used in the command.
	cmd := exec.CommandContext(ctx, d.VersionCommand[0], d.VersionCommand[1:]...)
	output, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(buf, "  ❌ Failed to get version: %v\n", err)
		return
	}

	version, err := d.extractVersion(output)
	if err != nil {
		fmt.Fprintf(buf, "  ❌ Could not determine version from output: %v\n", err)
		return
	}

	if err := d.validateVersion(version); err != nil {
		wantedVersion := fmt.Sprintf(d.VersionMatcher, version)
		if d.Optional {
			fmt.Fprintf(buf, "  ⚠️  %s is optional but should be upgraded: (have) %s (need)\n", d.Name, wantedVersion)
			return
		}
		fmt.Fprintf(buf, "  ❌ Needs upgrading: (have) %s (need)\n", wantedVersion)
		return
	}

	fmt.Fprintf(buf, "  ✅ Found version %s\n", version)
}

func (d *dependency) validateVersion(version string) error {
	if d.VersionMatcher == "" {
		return nil // No version matcher to validate against.
	}

	matcher := fmt.Sprintf(d.VersionMatcher, version)
	prog, err := expr.Compile(matcher)
	if err != nil {
		return fmt.Errorf("failed to compile version matcher %q: %w", matcher, err)
	}
	res, err := expr.Run(prog, nil)
	if err != nil {
		return fmt.Errorf("failed to evaluate version matcher %q: %w", matcher, err)
	}
	if res == nil || !dry.As[bool](res) {
		return fmt.Errorf("installed version %q does not match required version matcher %q", version, matcher)
	}
	return nil
}

func (d *dependency) extractVersion(output []byte) (string, error) {
	// Loop over each word in the output and attempt to parse out a version.
	var version string
	scan := bufio.NewScanner(bytes.NewReader(output))
	scan.Split(bufio.ScanWords)
	for scan.Scan() {
		word := scan.Text()
		if word == "" {
			continue // Skip empty words.
		}
		word = versionre.FindString(word)
		if word == "" {
			continue // Skip words that do not match the version regex.
		}
		// If the word starts with a digit prepend 'v' to it.
		if word[0] >= '0' && word[0] <= '9' {
			word = "v" + word // Prepend 'v' to the version.
		}
		if semver.IsValid(word) {
			version = word[1:] // Remove the 'v' prefix if it exists.
			break
		}
	}
	if err := scan.Err(); !errors.Is(err, io.EOF) && err != nil {
		return "", fmt.Errorf("failed to scan output: %w", err)
	}

	if version == "" {
		return "", fmt.Errorf("could not extract a valid version from output: %s", output)
	}
	return version, nil
}

func init() {
	v := internal.ValidatorFromContext(context.Background())
	var err error
	for _, dep := range cribDependencies {
		err = errors.Join(err, v.Struct(&dep))
	}
	if err != nil {
		panic(fmt.Errorf("cribctl doctor: failed to validate dependencies: %w", err))
	}
}
