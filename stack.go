package gortree

type Stack[T any] struct {
	items []T
}

// NewStack creates a new Stack
func NewStack[T any]() *Stack[T] {
	return &Stack[T]{}
}

// NewStackFrom creates a new Stack with initial items.
func NewStackFrom[T any](items ...T) *Stack[T] {
	stack := &Stack[T]{
		items: items,
	}
	return stack
}

// Len returns the number of items of the stack.
func (s *Stack[T]) Len() int {
	return len(s.items)
}

// Empty tells whether the stack does not have any items.
func (s *Stack[T]) Empty() bool {
	return len(s.items) == 0
}

// Push adds a new element to the stack.
func (s *Stack[T]) Push(e ...T) {
	s.items = append(s.items, e...)
}

// Pop returns the last element and removes it from the stack.
func (s *Stack[T]) Pop() (T, bool) {
	var zero T
	if len(s.items) == 0 {
		return zero, false
	}
	lastIdx := len(s.items) - 1
	item := s.items[lastIdx]
	s.items = s.items[:lastIdx]
	return item, true
}

// Peek returns the last element without modifying the stack.
func (s *Stack[T]) Peek() (T, bool) {
	var zero T
	if len(s.items) == 0 {
		return zero, false
	}
	return s.items[len(s.items)-1], true
}
