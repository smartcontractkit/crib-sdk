package service

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"
	"github.com/samber/lo"
	"github.com/xlab/treeprint"

	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/adapter/clientsideapply"
	"github.com/smartcontractkit/crib-sdk/internal/adapter/filehandler"
	"github.com/smartcontractkit/crib-sdk/internal/adapter/plancache"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/infra"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
	"github.com/smartcontractkit/crib-sdk/internal/core/port"
	"github.com/smartcontractkit/crib-sdk/internal/core/service/iresolver"
)

// mu is a package-level mutex to ensure that only one plan is being created at a time.
// This is necessary because cdk8s has some level of globally shared state, causing concurrent
// App and Chart creations to fail.
var mu sync.Mutex

type (
	builder interface {
		Build() port.Planner
	}

	// PlanService is a service that handles discovery of manifests in a directory, and applying them.
	// It includes the logic for applying special ClientSideApply manifests.
	//
	// The typical use of the plan service is:
	//
	//	// Create a new PlanService.
	//	svc := NewPlanService(ctx)
	//	// Create a new App Plan.
	//	appPlan, err := svc.CreatePlan(ctx, plan) // plan is a crib.Plan.
	//	// At this point, the DAG is available for inspection.
	//	// Synthesize the app, which will create the manifests in the tempdir.
	//	appPlan.Synthesize()
	//	// Finally, apply the plan, which will discover the manifests in the tempdir and apply them.
	//	err = svc.ApplyPlan(ctx)
	PlanService struct {
		fh port.FileHandler
	}

	// Manifest represents a manifest file with its name and whether it's purpose is
	// to be applied locally or remotely - ie ClientSideApply.
	Manifest struct {
		Name    string
		IsLocal bool
	}

	// ManifestBundle is a slice of Manifest, representing a bundle of manifests that can be applied together.
	ManifestBundle struct {
		root      string
		manifests []Manifest
		isLocal   bool
	}

	// AppPlan is a struct that represents the application plan, it includes
	// the root plan, the app, and the root chart.
	AppPlan struct {
		svc         *PlanService // svc is the PlanService that created this AppPlan.
		RootPlan    port.Planner
		App         cdk8s.App
		Chart       cdk8s.Chart
		planResults *plancache.Results
	}

	PlanState struct {
		*plancache.Results
	}
)

// NewPlanService creates a new PlanService with the provided FileHandler.
func NewPlanService(_ context.Context, fh *filehandler.Handler) (*PlanService, error) {
	return &PlanService{
		fh: fh,
	}, nil
}

// CreatePlan creates a new container app, and builds the plan with the provided components.
// An *AppPlan is returned, which will be ready to be synthesized, or any error that occurred.
// Note: All errors are collected and returned at once to allow for better debugging.
func (p *PlanService) CreatePlan(ctx context.Context, plan port.Planner) (*AppPlan, error) {
	// Create a new root app and chart for the plan.
	app := p.createApp(plan)
	ctx = internal.ContextWithConstruct(ctx, app.Chart)

	// Build the plan to resolve all dependencies, child plans, and detect cycles.
	// Note: The plan is recursively resolved so we don't need to build each child plan.
	planBuilder, ok := plan.(builder)
	if !ok {
		return nil, fmt.Errorf("plan %q does not implement Build() method", plan.Name())
	}
	app.RootPlan = planBuilder.Build()

	// cdk8s has some level of globally shared state, so we need to acquire a lock.
	mu.Lock()
	defer mu.Unlock()

	var resolutionErrors error
	// Loop through each child plan and their components, resolve them, and add them to the chart.
	for _, child := range app.RootPlan.ChildPlans() {
		for _, fn := range child.Components() {
			component, err := fn(ctx)
			if err != nil {
				resolutionErrors = errors.Join(resolutionErrors, err)
				continue
			}
			app.planResults.Add(component)
		}
	}
	// Loop through the parent components, resolve them, and add them to the chart.
	for _, fn := range app.RootPlan.Components() {
		component, err := fn(ctx)
		if err != nil {
			resolutionErrors = errors.Join(resolutionErrors, err)
			continue
		}
		app.planResults.Add(component)
	}
	// If there were any errors, return them.
	if resolutionErrors != nil {
		return nil, resolutionErrors
	}

	// Synthesize the app to create the manifests in the tempdir.
	app.App.Synth()
	return app, nil
}

// Apply applies the discovered manifests in the directory.
func (a *AppPlan) Apply(ctx context.Context) (*PlanState, error) {
	manifests := a.svc.findManifests()
	bundles := a.svc.normalizeManifests(manifests)
	for _, bundle := range bundles {
		err := bundle.Apply(ctx, a.svc)
		if errors.Is(err, domain.ErrAbort) {
			return nil, err
		}
		// TODO: Log errors with a Continue type.
	}
	return &PlanState{Results: a.planResults}, nil
}

// Apply creates a new runner and applies the manifest.
func (b ManifestBundle) Apply(ctx context.Context, p *PlanService) error {
	// Acquire a lock to ensure that only one client-side apply is being executed at a time.
	mu.Lock()
	defer mu.Unlock()

	var (
		runner port.ClientSideApplyRunner
		err    error
		m      *domain.ClientSideApplyManifest
	)

	m, err = b.Client(p)
	if err != nil {
		return domain.NewAbortError(fmt.Errorf("failed to create ClientSideApplyManifest for bundle %s: %w", b.String(), err))
	}
	runner, err = clientsideapply.NewRunner(m)
	if err != nil {
		return domain.NewAbortError(err)
	}
	_, err = runner.Execute(ctx, m)
	return dry.Wrapf(m.NewError(err), "unable to execute client-side apply for bundle %s", b.String())
}

func (b ManifestBundle) Client(p *PlanService) (*domain.ClientSideApplyManifest, error) {
	// If the bundle is a local manifest bundle, we return a ClientSideApplyManifest.
	if b.isLocal {
		return b.clientSideApply(p)
	}
	// Otherwise, we return a kubectlApply manifest.
	return b.kubectlApply()
}

// clientSideApply reads the manifest bundle and returns a ClientSideApplyManifest.
// It errors if the Bundle contains more than one manifest, or if the manifest is not a local manifest.
func (b ManifestBundle) clientSideApply(p *PlanService) (*domain.ClientSideApplyManifest, error) {
	if !b.isLocal {
		return nil, fmt.Errorf("bundle %s is not a local manifest bundle", b.String())
	}
	if len(b.manifests) != 1 {
		return nil, fmt.Errorf("bundle %s contains more than one manifest, expected 1", b.String())
	}

	// Read the manifest and unmarshal it into a domain.ClientSideApplyManifest.
	raw, err := p.fh.ReadFile(b.manifests[0].Name)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest %s: %w", b.manifests[0].Name, err)
	}
	m, err := domain.UnmarshalManifest[domain.ClientSideApplyManifest](raw)
	return dry.Wrapf2(m, err, "failed to unmarshal manifest %s", b.manifests[0].Name)
}

// kubectlApply creates a new ClientSideApplyManifest for the given non-local ManifestBundle.
func (b ManifestBundle) kubectlApply() (*domain.ClientSideApplyManifest, error) {
	if b.isLocal {
		return nil, fmt.Errorf("bundle %s is a local manifest bundle, expected remote", b.String())
	}
	if len(b.manifests) == 0 {
		return nil, fmt.Errorf("bundle %s contains no manifests", b.String())
	}

	// Create a new ClientSideApplyManifest for the kubectl apply command.
	return &domain.ClientSideApplyManifest{
		Spec: domain.ClientSideApplySpec{
			OnFailure: domain.FailureAbort,
			Action:    domain.ActionKubectl,
			Args: []string{
				"apply",
				"-f", b.String(),
				"--wait",
			},
		},
	}, nil
}

// createApp initializes a new basic application with directives on how to synthesize
// the resulting manifests.
func (p *PlanService) createApp(plan port.Planner) *AppPlan {
	// cdk8s has some level of globally shared state, so we need to acquire a lock.
	mu.Lock()
	defer mu.Unlock()

	// Create the resolvers.
	resolvers := append(
		plan.Resolvers(), // Plan resolvers.
		// Default resolvers.
		[]cdk8s.IResolver{
			iresolver.NewResolver(iresolver.NameResolver, iresolver.ResolutionPriorityLow),
		}...,
	)

	app := cdk8s.NewApp(&cdk8s.AppProps{
		Outdir:         dry.ToPtr(p.fh.Name()),
		YamlOutputType: cdk8s.YamlOutputType_FOLDER_PER_CHART_FILE_PER_RESOURCE,
		Resolvers:      dry.ToPtr(resolvers),
	})
	chart := cdk8s.NewChart(app, infra.ResourceID(plan.Name()+"."+plan.Namespace(), nil), &cdk8s.ChartProps{
		Namespace: dry.ToPtr(plan.Namespace()),
	})
	return &AppPlan{
		svc:         p,
		App:         app,
		Chart:       chart,
		planResults: plancache.New(),
	}
}

// findManifests uses the provided FileReader to scan a directory for manifest files.
// It returns a map where the keys are the path to the manifest files and the values
// are the list of matching files in that directory.
func (p *PlanService) findManifests() map[string][]Manifest {
	manifests := make(map[string][]Manifest)
	for path := range p.fh.Scan(p.discoverYAML) {
		manifests[filepath.Dir(path)] = append(manifests[filepath.Dir(path)], Manifest{
			Name:    path,
			IsLocal: p.isLocalManifest(path),
		})
	}
	return manifests
}

// normalizeManifests maps a map[string][]Manifest to a []ManifestBundle. It groups together
// the sets of manifests that can be applied together. For example, given the set:
//
//	 map[string][]Manifest{
//		"00": {
//			{Name: "00/00-namespace.yaml"},
//			{Name: "00/01-deployment.yaml"},
//			{Name: "00/02-client-side-apply.yaml", IsLocal: true},
//		"01": {
//			{Name: "01/00-service.yaml"},
//			{Name: "01/01-configmap.yaml"},
//		},
//	}
//
//		It will return the following slice of slices in 3 separate groups:
//
//		[]ManifestBundle{
//			// Group 1: All non-local manifests in the "00" directory.
//			{
//				{Name: "00/00-namespace.yaml"},
//				{Name: "00/01-deployment.yaml"},
//			},
//			// Group 2: The local manifest in the "00" directory.
//			{
//				{Name: "00/02-client-side-apply.yaml", IsLocal: true},
//			},
//			// Group 3: All non-local manifests in the "01" directory.
//			{
//				{Name: "01/00-service.yaml"},
//				{Name: "01/01-configmap.yaml"},
//			},
//		}
func (p *PlanService) normalizeManifests(manifests map[string][]Manifest) []ManifestBundle {
	var bundles []ManifestBundle
	for _, dir := range slices.Sorted(maps.Keys(manifests)) {
		currentBundle := ManifestBundle{
			root: p.fh.Name(),
		}
		var isLocal bool
		for _, m := range manifests[dir] {
			// If we are switching from a local to a non-local manifest, we need to close the current bundle.
			if isLocal != m.IsLocal {
				if len(currentBundle.manifests) > 0 {
					bundles = append(bundles, currentBundle)
				}
				// Reset the current bundle to start a new one.
				currentBundle = ManifestBundle{
					root:    p.fh.Name(),
					isLocal: m.IsLocal,
				}
				isLocal = m.IsLocal
			}
			// If the manifest is not local, add it to the current bundle manifests.
			if !m.IsLocal {
				currentBundle.manifests = append(currentBundle.manifests, m)
				continue
			}

			// The bundle is local. First close the current bundle.
			if len(currentBundle.manifests) > 0 {
				bundles = append(bundles, currentBundle)
			}
			// Append the local manifest to the current bundle.
			currentBundle.manifests = []Manifest{m}
		}
		// If there are any remaining manifests in the current bundle, we add it to the bundles.
		if len(currentBundle.manifests) > 0 {
			bundles = append(bundles, currentBundle)
		}
	}
	return bundles
}

// isLocalManifest checks whether the given manifest path is a local manifest by reading
// and unmarshaling the manifest, returning true if the version and kind represent a local
// manifest type.
func (p *PlanService) isLocalManifest(path string) bool {
	m, err := p.readManifest(path)
	if err != nil {
		return false
	}
	if m.APIVersion == domain.CribAPIVersion && m.Kind == domain.ClientSideApply {
		return true
	}
	return false
}

// readManifest reads the manifest file at the given path and unmarshals it into a domain.Manifest.
func (p *PlanService) readManifest(path string) (*domain.Manifest, error) {
	raw, err := p.fh.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading manifest %s: %w", path, err)
	}
	m, err := domain.UnmarshalManifest[domain.Manifest](raw)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling manifest %s: %w", path, err)
	}
	return m, nil
}

// discoverYAML checks if the given path is a yaml file, just by looking at the file extension.
func (*PlanService) discoverYAML(_ port.FileReader, path string) error {
	const (
		yamlFile = ".yaml"
		ymlFile  = ".yml"
	)

	ext := filepath.Ext(path)
	if strings.EqualFold(ext, yamlFile) || strings.EqualFold(ext, ymlFile) {
		return nil
	}
	return domain.NewSkipFileError(path)
}

// String returns a string representation of the ManifestBundle, which is a slice of Manifest.
func (b ManifestBundle) String() string {
	ss := lo.Map(b.manifests, func(m Manifest, _ int) string {
		return filepath.Join(b.root, m.Name)
	})
	return strings.Join(ss, ",")
}

// Preview renders the DAG as a tree structure and returns it as a string.
func (a *AppPlan) Preview(ctx context.Context) string {
	tree := treeprint.New()

	// Create a dummy CDK8s app/chart for context
	app := cdk8s.NewApp(nil)
	chartName := "preview"
	chart := cdk8s.NewChart(app, &chartName, nil)
	ctx = internal.ContextWithConstruct(ctx, chart)

	// Build tree based on plan structure rather than CDK8s constructs
	rootBranch := tree.AddBranch(fmt.Sprintf("%s.%s", a.RootPlan.Name(), a.RootPlan.Namespace()))

	// Cache for resolved components to avoid duplicate construction
	resolvedComponents := make([]port.Component, 0)
	totalComponents := 0

	// Add child plans first
	for _, childPlan := range a.RootPlan.ChildPlans() {
		childBranch := rootBranch.AddBranch(fmt.Sprintf("Plan: %s.%s", childPlan.Name(), childPlan.Namespace()))
		for _, componentFn := range childPlan.Components() {
			// Resolve the component once and cache it
			component, err := componentFn(ctx)
			if err != nil {
				childBranch.AddNode(fmt.Sprintf("<error: %v>", err))
				continue
			}
			if component == nil {
				childBranch.AddNode("<nil component>")
				continue
			}

			// Cache the component for later counting
			resolvedComponents = append(resolvedComponents, component)

			// Get display name
			name := getComponentDisplayNameFromComponent(component)
			componentBranch := childBranch.AddBranch(name)
			addNestedComponents(ctx, componentBranch, component)
		}
	}

	// Add root plan components
	for _, componentFn := range a.RootPlan.Components() {
		// Resolve the component once and cache it
		component, err := componentFn(ctx)
		if err != nil {
			rootBranch.AddNode(fmt.Sprintf("<error: %v>", err))
			continue
		}
		if component == nil {
			rootBranch.AddNode("<nil component>")
			continue
		}

		// Cache the component for later counting
		resolvedComponents = append(resolvedComponents, component)

		// Get display name
		name := getComponentDisplayNameFromComponent(component)
		componentBranch := rootBranch.AddBranch(name)
		addNestedComponents(ctx, componentBranch, component)
	}

	// Count only the displayed nested components
	for _, component := range resolvedComponents {
		totalComponents += countDisplayedComponents(ctx, component)
	}

	// Add summary information
	var summary strings.Builder
	summary.WriteString(tree.String())
	fmt.Fprintf(&summary, "\nSummary:\n")
	fmt.Fprintf(&summary, "- Root Plan: %s.%s\n", a.RootPlan.Name(), a.RootPlan.Namespace())
	fmt.Fprintf(&summary, "- Root Components: %d\n", len(a.RootPlan.Components()))
	fmt.Fprintf(&summary, "- All Nested Components: %d\n", totalComponents)

	return summary.String()
}

// addNestedComponents recursively adds nested components to the tree.
func addNestedComponents(ctx context.Context, branch treeprint.Tree, component port.Component) {
	// Get the component's node to access its children
	if component.Node() == nil {
		return
	}

	// Try to get children from the component
	children := component.Node().Children()
	if children == nil || len(*children) == 0 {
		return
	}

	// Add each child component
	for _, child := range *children {
		if child == nil {
			continue
		}

		// Get the child's ID for display
		var childName string
		if child.Node() != nil && child.Node().Id() != nil {
			childName = *child.Node().Id()
		} else {
			childName = fmt.Sprintf("%T", child)
		}

		// Add the child as a node (not a branch since we can't easily recurse into IConstruct)
		branch.AddNode(childName)
	}
}

// getComponentDisplayNameFromComponent gets the display name from an already resolved component.
func getComponentDisplayNameFromComponent(component port.Component) string {
	if component == nil {
		return "<nil component>"
	}
	if component.Node() != nil && component.Node().Id() != nil {
		id := component.Node().Id()
		return fmt.Sprintf("%s (%T)", *id, component)
	}
	return fmt.Sprintf("%T", component)
}

// Count only the immediate children that are displayed as nodes in the tree.
func countDisplayedComponents(ctx context.Context, component port.Component) int {
	if component == nil || component.Node() == nil {
		return 0
	}

	children := component.Node().Children()
	if children == nil || len(*children) == 0 {
		return 0 // Root components aren't counted as "nested"
	}

	count := 0
	for _, child := range *children {
		if child == nil {
			continue
		}
		// Only count immediate children that would be displayed as nodes
		// This matches what addNestedComponents displays
		count++
	}
	return count
}
