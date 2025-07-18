package plancache

import (
	"context"
	"maps"
	"slices"
	"testing"
	"unique"

	"github.com/aws/constructs-go/constructs/v10"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/suite"

	"github.com/smartcontractkit/crib-sdk/internal/core/common/infra"
)

func TestCacheSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(CacheSuite))
}

type testProps map[string]any

func (testProps) Validate(ctx context.Context) error {
	// No validation needed for test props
	return nil
}

type CacheSuite struct {
	suite.Suite

	results *Results

	root         constructs.Construct
	child1       constructs.Construct
	subchild1    constructs.Construct
	subsubchild1 constructs.Construct
	child2       constructs.Construct
}

func (s *CacheSuite) SetupSuite() {
	props := func() testProps {
		return map[string]any{
			gofakeit.Adjective(): gofakeit.BuzzWord(),
		}
	}

	// Create the following structure:
	// root
	// root/child1
	// root/child1/subchild1
	// root/child1/subchild1/child1
	// root/child2

	s.root = constructs.NewRootConstruct(infra.ResourceID("root", props()))
	s.child1 = constructs.NewConstruct(s.root, infra.ResourceID("child1", props()))
	s.subchild1 = constructs.NewConstruct(s.child1, infra.ResourceID("subchild1", props()))
	s.subsubchild1 = constructs.NewConstruct(s.subchild1, infra.ResourceID("child1", props()))
	s.child2 = constructs.NewConstruct(s.root, infra.ResourceID("child2", props()))
}

func (s *CacheSuite) SetupTest() {
	// Reset the results before each test
	s.results = New()
}

func (s *CacheSuite) Test_parentID() {
	tests := []struct {
		name string
		node constructs.Construct
		want string
	}{
		{
			name: "nil node",
			node: nil,
			want: "",
		},
		{
			name: "root node",
			node: s.root,
			want: "",
		},
		{
			name: "child1 node",
			node: s.child1,
			want: "root",
		},
		{
			name: "subchild1 node",
			node: s.subchild1,
			want: "child1",
		},
		{
			name: "subsubchild1 node",
			node: s.subsubchild1,
			want: "subchild1",
		},
		{
			name: "child2 node",
			node: s.child2,
			want: "root",
		},
	}
	for _, tc := range tests {
		s.Run(tc.name, func() {
			got := parentID(tc.node)
			s.Equal(tc.want, got.Value())
		})
	}
}

func (s *CacheSuite) Test_Results_Add() {
	s.results.Add(s.root)
	s.results.Add(s.child1)
	s.results.Add(s.subchild1)
	s.results.Add(s.subsubchild1) // Key is "child1" because of the path.
	s.results.Add(s.child2)

	// Assertions
	s.Len(s.results.nodes, 4, "Expected 4 unique resource IDs - root, child1, subchild1, child2")
	s.Len(s.results.roots, 5, "Expected 5 root nodes in results")
	s.Len(slices.Collect(s.results.Get("bogus")), 0, "Expected no nodes for bogus resource ID")
	s.Len(slices.Collect(s.results.Get("root")), 1)
	s.Len(slices.Collect(s.results.Get("child1")), 2)
	s.Len(slices.Collect(s.results.Get("subchild1")), 1)
	s.Len(slices.Collect(s.results.Get("child2")), 1)

	for key := range maps.Keys(s.results.nodes) {
		s.T().Logf("Key: %s", key.Value())
	}
}

func (s *CacheSuite) Test_Results_Get() {
	s.results.Add(s.child1)
	s.results.Add(s.subsubchild1)

	results := slices.Collect(s.results.Get("child1"))
	s.Require().Len(results, 2, "Expected 2 nodes for 'child1' resource ID")

	s.Equal(&Node{
		ID:       unique.Make[string]("child1"),
		IDStr:    "child1",
		data:     s.child1,
		ParentID: unique.Make[string]("root"),
	}, results[0], "First node should be child1 with parent root")

	s.Equal(&Node{
		ID:       unique.Make[string]("child1"),
		IDStr:    "child1",
		data:     s.subsubchild1,
		ParentID: unique.Make[string]("subchild1"),
	}, results[1], "Second node should be subsubchild1 with parent subchild1")
}

func (s *CacheSuite) Test_Results_Components() {
	s.results.Add(s.root)
	s.results.Add(s.child1)
	s.results.Add(s.subchild1)
	s.results.Add(s.subsubchild1) // Key is "child1" because of the path.
	s.results.Add(s.child2)

	components := slices.Collect(s.results.Components())
	s.Require().Len(components, 5, "Expected 5 components in results")
	s.Equal(components[0], s.root, "First component should be root")
	s.Equal(components[1], s.child1, "Second component should be child1")
	s.Equal(components[2], s.subchild1, "Third component should be subchild1")
	s.Equal(components[3], s.subsubchild1, "Fourth component should be subsubchild1")
	s.Equal(components[4], s.child2, "Fifth component should be child2")
}
