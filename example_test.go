package terrors_test

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sort"

	"github.com/lukasschwab/terrors"
)

// ExampleWalk_leaves demonstrates collecting and classifying every leaf error
// in a tree of wrapped and joined errors using [terrors.Walk].
//
// A "leaf" is an error that does not itself wrap any other errors. Nil leaves
// are treated as caller policy: this example skips them.
func ExampleWalk_leaves() {
	// Build a tree of errors.
	timeout := &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("i/o timeout")}
	permission := fmt.Errorf("open config: %w", errors.New("permission denied"))
	root := errors.Join(
		fmt.Errorf("network: %w", timeout),
		permission,
	)

	// Collect leaves with a Visitor.
	collector := &classifyingVisitor{}
	terrors.Walk(collector, root)

	// Print the classified leaves.
	sort.Strings(collector.descriptions)
	for _, d := range collector.descriptions {
		fmt.Println(d)
	}

	// Output:
	// leaf: "i/o timeout" (other)
	// leaf: "permission denied" (other)
}

// classifyingVisitor collects leaf errors and classifies them. It tracks
// whether the current node has children by checking the Unwrap interfaces.
type classifyingVisitor struct {
	descriptions []string
}

func (cv *classifyingVisitor) Visit(err error) terrors.Visitor {
	if err == nil {
		return nil // skip nil leaves
	}

	// Check if this error has children (i.e. wraps other errors).
	type errorGroup interface{ Unwrap() []error }
	type errorWrapper interface{ Unwrap() error }
	switch err.(type) {
	case errorGroup, errorWrapper:
		// Not a leaf: keep walking.
		return cv
	}

	// This is a leaf: classify it.
	kind := "other"
	if errors.Is(err, io.EOF) {
		kind = "EOF"
	}
	cv.descriptions = append(cv.descriptions, fmt.Sprintf("leaf: %q (%s)", err.Error(), kind))

	// No need to descend further.
	return nil
}

// ExampleWalk_search demonstrates short-circuiting a tree walk once a matching
// error is found, implementing an Any(err, match) bool pattern.
func ExampleWalk_search() {
	target := errors.New("secret-error")

	// Build a tree where the target is buried.
	root := errors.Join(
		fmt.Errorf("unrelated: %w", errors.New("noise")),
		fmt.Errorf("wrapper: %w",
			errors.Join(
				errors.New("also noise"),
				fmt.Errorf("deep: %w", target),
			),
		),
	)

	found := Any(root, func(err error) bool {
		return errors.Is(err, target)
	})
	fmt.Println("found:", found)

	missing := Any(root, func(err error) bool {
		return errors.Is(err, io.EOF)
	})
	fmt.Println("missing:", !missing)

	// Output:
	// found: true
	// missing: true
}

// Any reports whether any error in the tree rooted at err satisfies the match
// predicate. It short-circuits as soon as a match is found.
func Any(err error, match func(error) bool) bool {
	searcher := &searchVisitor{match: match}
	terrors.Walk(searcher, err)
	return searcher.found
}

// searchVisitor implements [terrors.Visitor] with early termination. Once a
// match is found, Visit returns nil to stop descending.
type searchVisitor struct {
	match func(error) bool
	found bool
}

func (sv *searchVisitor) Visit(err error) terrors.Visitor {
	if sv.found {
		return nil // already matched; stop traversal.
	}
	if sv.match(err) {
		sv.found = true
		return nil // short-circuit: no need to visit children.
	}
	return sv
}
