package plancache

import (
	"iter"
	"slices"
	"strings"
	"sync"
	"unique"

	"github.com/aws/constructs-go/constructs/v10"

	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/infra"
)

type (
	// Node represents a single node in the results tree.
	Node struct {
		data     constructs.IConstruct
		ID       unique.Handle[string]
		ParentID unique.Handle[string]
		IDStr    string
	}

	// Results manages fast lookups and hierarchical relations.
	Results struct {
		nodes map[unique.Handle[string]][]*Node
		roots []*Node

		mu sync.RWMutex
	}
)

// New initializes a new Results instance with an empty map.
func New() *Results {
	return &Results{
		nodes: make(map[unique.Handle[string]][]*Node),
	}
}

// Add inserts a new node into the results tree.
func (r *Results) Add(c constructs.IConstruct) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if c == nil || c.Node() == nil || c.Node().Id() == nil {
		return
		// TODO(polds): handle this case more gracefully, maybe return an error or log a warning.
	}

	id := prepareID(c.Node().Id())
	idStr := infra.ExtractResource(c.Node().Id())
	node := &Node{
		ID:       id,
		IDStr:    idStr,
		data:     c,
		ParentID: parentID(c),
	}
	r.nodes[id] = append(r.nodes[id], node)
	r.roots = append(r.roots, node)
}

// Get retrieves nodes by their ID.
func (r *Results) Get(resource string) iter.Seq[*Node] {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id := unique.Make[string](resource)
	return slices.Values(r.nodes[id])
}

// Components returns an iterator over all components in the order that they were added.
func (r *Results) Components() iter.Seq[constructs.IConstruct] {
	return func(yield func(constructs.IConstruct) bool) {
		r.mu.RLock()
		defer r.mu.RUnlock()

		for _, node := range r.roots {
			if !yield(node.Component()) {
				return
			}
		}
	}
}

// Component returns the component associated with the node.
func (n *Node) Component() constructs.IConstruct {
	if n == nil {
		return nil
	}
	return n.data
}

// Children returns all child nodes of the given node.
func (r *Results) Children(parent *Node) []*Node {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var children []*Node
	for _, node := range r.roots {
		if node.ParentID == parent.ID {
			children = append(children, node)
		}
	}
	return children
}

// RootNodes returns all root nodes (nodes without parents).
func (r *Results) RootNodes() []*Node {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var roots []*Node
	for _, node := range r.roots {
		if node.ParentID == unique.Make[string]("") {
			roots = append(roots, node)
		}
	}
	return roots
}

// String returns a string representation of the node ID.
func (n *Node) String() string {
	if n == nil {
		return ""
	}
	return n.IDStr
}

// parentID determines the parent node ID for a given component.
// It splits the component's path and finds the parent node ID based on the path segments.
func parentID(c constructs.IConstruct) unique.Handle[string] {
	if c == nil || c.Node() == nil || c.Node().Id() == nil || c.Node().Path() == nil {
		return unique.Make[string]("")
	}

	// Determine the parent node based on the component's path.
	var (
		currentID = dry.FromPtr(c.Node().Id())
		parentID  string
	)

	for path := range pathSplitFn(*c.Node().Path()) {
		if path == currentID {
			break
		}
		parentID = path
	}
	// Prepare the parent ID.
	if parentID == "" {
		return unique.Make[string]("")
	}

	return prepareID(dry.ToPtr(parentID))
}

func prepareID(resource *string) unique.Handle[string] {
	id := infra.ExtractResource(resource)
	return unique.Make[string](id)
}

func pathSplitFn(path string) iter.Seq[string] {
	return strings.FieldsFuncSeq(path, func(r rune) bool {
		return r == '/'
	})
}
