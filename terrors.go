// terrors is a utility for walking trees of wrapped and joined errors. terrors
// is modeled on the standard library's AST package: see [ast.Walk].
//
// [ast.Walk]: https://pkg.go.dev/go/ast#Walk
package terrors

// A Visitor controls traversal of an error tree at a given error node.
type Visitor interface {
	// Visit err. This is called for err before any of err's children are
	// walked. If Visit returns nil, the children are skipped; otherwise,
	// the children are visited with the returned Visitor w.
	//
	// In many cases, Visit will return the original visitor unchanged to apply
	// the same stateless behavior to all error nodes in the tree.
	Visit(err error) (w Visitor)
}

// Walk is the equivalent of calling `Tree{err}.Walk(v)`.
func Walk(v Visitor, err error) {
	Tree{err}.Walk(v)
}

type Tree struct {
	Err error
}

// Walk traverses the tree in depth-first order, starting from the root. If the
// visitor w returned by calling v.Visit on the root of the tree is not nil,
// Walk is invoked recursively with w for each child of the root.
func (n Tree) Walk(v Visitor) {
	// Visit this node.
	w := v.Visit(n.Err)
	// Exit early if the visitor returned nil.
	if w == nil {
		return
	}

	// Walk children.
	var children []error
	if group, ok := n.Err.(errorGroup); ok {
		children = group.Unwrap()
	} else if parent, ok := n.Err.(errorWrapper); ok {
		children = []error{parent.Unwrap()}
	}
	for _, child := range children {
		Tree{child}.Walk(w)
	}
}

// An error e wraps another error if e's type has one of the methods [...]
// Unwrap() error. See the [errors] package.
//
// [errors]: https://pkg.go.dev/errors
type errorWrapper interface {
	Unwrap() error
}

// An error e wraps another error if e's type has one of the methods [...]
// Unwrap() []error. See the [errors] package.
//
// [errors]: https://pkg.go.dev/errors
type errorGroup interface {
	Unwrap() []error
}
