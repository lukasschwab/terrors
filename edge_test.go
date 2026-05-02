package terrors_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/lukasschwab/terrors"
	"github.com/peterldowns/testy/assert"
)

// --- Custom error types for edge cases ---

// groupError implements Unwrap() []error with caller-controlled children.
type groupError struct {
	msg      string
	children []error
}

func (e *groupError) Error() string    { return e.msg }
func (e *groupError) Unwrap() []error  { return e.children }

// wrapError implements Unwrap() error with a caller-controlled inner error.
type wrapError struct {
	msg   string
	inner error
}

func (e *wrapError) Error() string { return e.msg }
func (e *wrapError) Unwrap() error { return e.inner }

// pruningVisitor returns nil for any error matching pruneAt, stopping descent.
type pruningVisitor struct {
	visited []error
	pruneAt error
}

func (pv *pruningVisitor) Visit(err error) terrors.Visitor {
	pv.visited = append(pv.visited, err)
	if err == pv.pruneAt {
		return nil
	}
	return pv
}

// --- Edge case tests ---

// errors.Join() returns nil; walking nil visits a single nil node.
func TestWalk_JoinNoArgs_ReturnsNil(t *testing.T) {
	root := errors.Join() // nil

	visitor := &Appender{[]error{}}
	terrors.Walk(visitor, root)

	assert.Equal(t, 1, len(visitor.Visited))
	assert.Nil(t, visitor.Visited[0])
}

// errors.Join(nil, nil) also returns nil.
func TestWalk_JoinNilNil_ReturnsNil(t *testing.T) {
	root := errors.Join(nil, nil) // nil

	visitor := &Appender{[]error{}}
	terrors.Walk(visitor, root)

	assert.Equal(t, 1, len(visitor.Visited))
	assert.Nil(t, visitor.Visited[0])
}

// errors.Join(nil, err) filters out the nil child before terrors sees it.
// The stdlib join error only contains the non-nil error.
func TestWalk_JoinNilAndErr_StdlibFiltersNil(t *testing.T) {
	child := errors.New("child")
	root := errors.Join(nil, child)

	visitor := &Appender{[]error{}}
	terrors.Walk(visitor, root)

	// root + child (nil was filtered by errors.Join).
	assert.Equal(t, 2, len(visitor.Visited))
	assert.Equal(t, root, visitor.Visited[0], cmpopts.EquateErrors())
	assert.Equal(t, child, visitor.Visited[1], cmpopts.EquateErrors())
}

// errors.Join(err, nil) filters out the nil child.
func TestWalk_JoinErrAndNil_StdlibFiltersNil(t *testing.T) {
	child := errors.New("child")
	root := errors.Join(child, nil)

	visitor := &Appender{[]error{}}
	terrors.Walk(visitor, root)

	assert.Equal(t, 2, len(visitor.Visited))
	assert.Equal(t, root, visitor.Visited[0], cmpopts.EquateErrors())
	assert.Equal(t, child, visitor.Visited[1], cmpopts.EquateErrors())
}

// A custom Unwrap() []error returning an empty slice: no children are visited.
func TestWalk_GroupEmptySlice_NoChildren(t *testing.T) {
	root := &groupError{msg: "empty-group", children: []error{}}

	visitor := &Appender{[]error{}}
	terrors.Walk(visitor, root)

	assert.Equal(t, 1, len(visitor.Visited))
	assert.Equal(t, error(root), visitor.Visited[0], cmpopts.EquateErrors())
}

// A custom Unwrap() []error that includes nil entries: nil children are visited.
func TestWalk_GroupWithNilEntries_VisitsNils(t *testing.T) {
	child := errors.New("real")
	root := &groupError{msg: "has-nils", children: []error{nil, child, nil}}

	visitor := &Appender{[]error{}}
	terrors.Walk(visitor, root)

	// root, nil, child, nil
	assert.Equal(t, 4, len(visitor.Visited))
	assert.Equal(t, error(root), visitor.Visited[0], cmpopts.EquateErrors())
	assert.Nil(t, visitor.Visited[1])
	assert.Equal(t, child, visitor.Visited[2], cmpopts.EquateErrors())
	assert.Nil(t, visitor.Visited[3])
}

// A custom Unwrap() error that returns nil: the nil child is visited.
func TestWalk_WrapperReturnsNil_VisitsNilChild(t *testing.T) {
	root := &wrapError{msg: "wraps-nil", inner: nil}

	visitor := &Appender{[]error{}}
	terrors.Walk(visitor, root)

	assert.Equal(t, 2, len(visitor.Visited))
	assert.Equal(t, error(root), visitor.Visited[0], cmpopts.EquateErrors())
	assert.Nil(t, visitor.Visited[1])
}

// Returning nil from Visit for a non-root node prunes that subtree.
func TestWalk_PruneNonRoot_SkipsSubtree(t *testing.T) {
	leaf := errors.New("leaf")
	pruned := fmt.Errorf("pruned: %w", leaf)
	other := errors.New("other")
	root := errors.Join(pruned, other)

	visitor := &pruningVisitor{pruneAt: pruned}
	terrors.Walk(visitor, root)

	// root is visited, pruned is visited (then pruned), other is visited.
	// leaf is NOT visited because we pruned at "pruned".
	assert.Equal(t, 3, len(visitor.visited))
	assert.Equal(t, root, visitor.visited[0], cmpopts.EquateErrors())
	assert.Equal(t, pruned, visitor.visited[1], cmpopts.EquateErrors())
	assert.Equal(t, other, visitor.visited[2], cmpopts.EquateErrors())
}

// Depth-first order through a nested mix of Unwrap() []error and Unwrap() error.
//
//	root (group)
//	├── a (wrap)
//	│   └── b (leaf)
//	├── c (group)
//	│   ├── d (leaf)
//	│   └── e (wrap)
//	│       └── f (leaf)
//	└── g (leaf)
func TestWalk_NestedMix_DepthFirstOrder(t *testing.T) {
	b := errors.New("b")
	a := &wrapError{msg: "a", inner: b}
	d := errors.New("d")
	f := errors.New("f")
	e := &wrapError{msg: "e", inner: f}
	c := &groupError{msg: "c", children: []error{d, e}}
	g := errors.New("g")
	root := &groupError{msg: "root", children: []error{a, c, g}}

	visitor := &Appender{[]error{}}
	terrors.Walk(visitor, root)

	expected := []error{root, a, b, c, d, e, f, g}
	assert.Equal(t, len(expected), len(visitor.Visited))
	for i, want := range expected {
		assert.Equal(t, want, visitor.Visited[i], cmpopts.EquateErrors())
	}
}

// Verify that terrors.Walk delegates to Tree{}.Walk: both produce identical
// visit sequences.
func TestWalk_PackageFuncDelegatesToTreeWalk(t *testing.T) {
	inner := errors.New("inner")
	middle := fmt.Errorf("middle: %w", inner)
	root := errors.Join(middle, errors.New("sibling"))

	v1 := &Appender{[]error{}}
	terrors.Walk(v1, root)

	v2 := &Appender{[]error{}}
	terrors.Tree{Err: root}.Walk(v2)

	assert.Equal(t, len(v1.Visited), len(v2.Visited))
	for i := range v1.Visited {
		assert.Equal(t, v1.Visited[i], v2.Visited[i], cmpopts.EquateErrors())
	}
}

// nilVisitor always returns nil from Visit, pruning all children.
type nilVisitor struct {
	visited []error
}

func (nv *nilVisitor) Visit(err error) terrors.Visitor {
	nv.visited = append(nv.visited, err)
	return nil
}

// Pruning at the root: visitor returns nil for the root, so no children
// are visited at all.
func TestWalk_PruneAtRoot_NoChildrenVisited(t *testing.T) {
	child := errors.New("child")
	root := fmt.Errorf("root: %w", child)

	visitor := &nilVisitor{}
	terrors.Walk(visitor, root)

	assert.Equal(t, 1, len(visitor.visited))
	assert.Equal(t, root, visitor.visited[0], cmpopts.EquateErrors())
}
