package terrors_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/lukasschwab/terrors"
	"github.com/peterldowns/testy/assert"
)

type Appender struct {
	Visited []error
}

func (s *Appender) Visit(err error) terrors.Visitor {
	s.Visited = append(s.Visited, err)
	return s
}

func TestWalk_Nil(t *testing.T) {
	visitor := &Appender{[]error{}}
	terrors.Tree{nil}.Walk(visitor)

	assert.Equal(t, 1, len(visitor.Visited))
	assert.Nil(t, visitor.Visited[0])
}

func TestWalk_Single(t *testing.T) {
	err := errors.New("single")

	visitor := &Appender{[]error{}}
	terrors.Tree{err}.Walk(visitor)

	assert.Equal(t, 1, len(visitor.Visited))
	assert.Equal(t, err, visitor.Visited[0], cmpopts.EquateErrors())
}

func TestWalk_Wrap(t *testing.T) {
	inner := errors.New("single")
	middle := fmt.Errorf("middle: %w", inner)
	outer := fmt.Errorf("outer: %w", middle)

	visitor := &Appender{[]error{}}
	terrors.Tree{outer}.Walk(visitor)

	assert.Equal(t, 3, len(visitor.Visited))
	assert.Equal(t, outer, visitor.Visited[0], cmpopts.EquateErrors())
	assert.Equal(t, middle, visitor.Visited[1], cmpopts.EquateErrors())
	assert.Equal(t, inner, visitor.Visited[2], cmpopts.EquateErrors())
}

func TestWalk_Group(t *testing.T) {
	first := errors.New("first")
	second := errors.New("second")
	root := errors.Join(first, second)

	visitor := &Appender{[]error{}}
	terrors.Tree{root}.Walk(visitor)

	assert.Equal(t, 3, len(visitor.Visited))
	assert.Equal(t, root, visitor.Visited[0], cmpopts.EquateErrors())
	assert.Equal(t, first, visitor.Visited[1], cmpopts.EquateErrors())
	assert.Equal(t, second, visitor.Visited[2], cmpopts.EquateErrors())
}

func TestWalk_Compound(t *testing.T) {
	first := errors.New("first")
	inner := errors.New("single")
	outer := fmt.Errorf("outer: %w", inner)
	root := errors.Join(first, outer)

	visitor := &Appender{[]error{}}
	terrors.Tree{root}.Walk(visitor)

	assert.Equal(t, 4, len(visitor.Visited))
	assert.Equal(t, root, visitor.Visited[0], cmpopts.EquateErrors())
	assert.Equal(t, first, visitor.Visited[1], cmpopts.EquateErrors())
	assert.Equal(t, outer, visitor.Visited[2], cmpopts.EquateErrors())
	assert.Equal(t, inner, visitor.Visited[3], cmpopts.EquateErrors())
}

func TestWalk_WrapNil(t *testing.T) {
	root := fmt.Errorf("wrapped: %w", nil)

	visitor := &Appender{[]error{}}
	terrors.Tree{root}.Walk(visitor)

	assert.Equal(t, 2, len(visitor.Visited))
	assert.Equal(t, root, visitor.Visited[0], cmpopts.EquateErrors())
	assert.Nil(t, visitor.Visited[1])
}
