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

// SetCursor positions the iterator at a specific server-supplied cursor.
// Use to resume iteration across process boundaries (e.g. a paginated
// admin UI that returns the cursor in a URL query param). The next
// NextPage / Next call fetches starting from this position. Calling
// after iteration has already produced items has undefined behaviour;
// only call before the first Next/NextPage.
func (it *Iter[T]) SetCursor(cursor string) {
	if it == nil {
		return
	}
	it.cursor = cursor
	it.first = false
}

// NextPage fetches one page from the underlying API and returns the
// items together with the cursor for the next page. Empty cursor
// indicates the stream is exhausted. Unlike Next, callers see explicit
// page boundaries — useful for UIs that render one page at a time.
func (it *Iter[T]) NextPage(ctx context.Context) ([]T, string, error) {
	if it == nil {
		return nil, "", nil
	}
	if it.err != nil {
		return nil, "", it.err
	}
	if it.finished {
		return nil, "", nil
	}
	if !it.first && it.cursor == "" {
		it.finished = true
		return nil, "", nil
	}
	it.first = false

	items, next, err := it.fetch(ctx, it.cursor)
	if err != nil {
		it.err = err
		return nil, "", err
	}
	it.cursor = next
	if next == "" {
		it.finished = true
	}
	return items, next, nil
}
