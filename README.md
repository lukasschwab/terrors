# terrors [![Go Reference](https://pkg.go.dev/badge/github.com/lukasschwab/terrors.svg)](https://pkg.go.dev/github.com/lukasschwab/terrors)

`terrors` is a utility for walking trees of wrapped and joined errors. `terrors` is modeled on the standard library's AST package: see [`ast.Walk`](https://pkg.go.dev/go/ast#Walk).

For more on error composition, see the standard library's [`errors` package documentation](https://pkg.go.dev/errors).

[![](https://upload.wikimedia.org/wikipedia/commons/thumb/1/16/The_terrible_plague_of_locusts_in_Palestine%2C_March-June_1915._Locusts_denuding_a_fig_tree._LOC_matpc.01902.tif/lossy-page1-1280px-The_terrible_plague_of_locusts_in_Palestine%2C_March-June_1915._Locusts_denuding_a_fig_tree._LOC_matpc.01902.tif.jpg)](https://commons.wikimedia.org/wiki/File:The_terrible_plague_of_locusts_in_Palestine,_March-June_1915._Locusts_denuding_a_fig_tree._LOC_matpc.01902.tif)

## Motivation

Go errors can form trees, not just linear chains where each error wraps the
next. `errors.Join`, introduced in Go 1.20, is one example; so is any other
error implementing `Unwrap() []error`.

Use the standard `errors.Is`, `errros.As`, and `errors.AsType` to check whether
an error tree contains a known target or type.

`terrors.Walk` is for cases where matching is not enough and each visited error
needs to be exposed to caller-defined logic. Use `terrors` when you need to

- collect every error in a wrapped or joined error tree
- classify errors for logging, metrics, or reporting
- search with a custom predicate that is not expressible as `errors.Is`, etc.
- filter, summarize, or transform information from multiple branches of an
  error tree

For example, you might want to skip logging error trees where every leaf is
`context.Canceled`. `errors.Is` would lead you to miss meaningful errors you do
want to log anytime they are joined with at least one `context.Canceled`.

