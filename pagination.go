package graphann

import "context"

// Iter is a forward-only cursor over a paginated result set. Use:
//
//	it := client.ListJobs(ctx, ListJobsRequest{...})
//	for it.Next(ctx) {
//	    job := it.Item()
//	    // ...
//	}
//	if err := it.Err(); err != nil { ... }
type Iter[T any] struct {
	// fetch returns the next page given the previous cursor. Empty
	// nextCursor terminates iteration.
	fetch func(ctx context.Context, cursor string) (items []T, nextCursor string, err error)

	cursor   string
	first    bool
	finished bool

	buf  []T
	bufN int

	last T
	err  error
}

// newIter constructs an Iter. The first Next call invokes fetch with an
// empty cursor.
func newIter[T any](fetch func(ctx context.Context, cursor string) ([]T, string, error)) *Iter[T] {
	return &Iter[T]{fetch: fetch, first: true}
}

// Next advances the iterator to the next item. Returns false when the
// stream is exhausted or an error occurred (use Err to distinguish).
func (it *Iter[T]) Next(ctx context.Context) bool {
	if it == nil || it.err != nil || it.finished {
		return false
	}
	if it.bufN < len(it.buf) {
		it.last = it.buf[it.bufN]
		it.bufN++
		return true
	}
	// Need to fetch more.
	if !it.first && it.cursor == "" {
		it.finished = true
		return false
	}
	it.first = false

	items, next, err := it.fetch(ctx, it.cursor)
	if err != nil {
		it.err = err
		return false
	}
	it.buf = items
	it.bufN = 0
	it.cursor = next
	if len(items) == 0 {
		it.finished = true
		return false
	}
	it.last = it.buf[it.bufN]
	it.bufN++
	return true
}

// Item returns the current value. Defined behaviour only after a Next
// call that returned true.
func (it *Iter[T]) Item() T {
	if it == nil {
		var zero T
		return zero
	}
	return it.last
}

// Err returns the terminal error, if any. nil after a clean exhaustion.
func (it *Iter[T]) Err() error {
	if it == nil {
		return nil
	}
	return it.err
}

// All consumes the iterator and returns every item. Convenient for
// paged endpoints whose total result set is small.
func (it *Iter[T]) All(ctx context.Context) ([]T, error) {
	var out []T
	for it.Next(ctx) {
		out = append(out, it.Item())
	}
	return out, it.Err()
}
