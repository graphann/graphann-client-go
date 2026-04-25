package graphann

import (
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// sfGroup wraps singleflight.Group with a "sticky" window: once a key
// resolves, the result is held for a configurable window so additional
// callers within the window can observe the same response without
// hitting the network. This complements the LRU+TTL queryCache for
// dedup'ing concurrent identical search bursts.
//
// nil-receiver Do is safe and falls through to fn() directly.
type sfGroup struct {
	g      singleflight.Group
	window time.Duration

	mu      sync.Mutex
	pending map[string]*sfPending
}

type sfPending struct {
	value  any
	err    error
	expiry time.Time
}

// newSFGroup returns an sfGroup whose results stay sticky for window.
// window <= 0 disables singleflight entirely.
func newSFGroup(window time.Duration) *sfGroup {
	if window <= 0 {
		return nil
	}
	return &sfGroup{
		window:  window,
		pending: make(map[string]*sfPending),
	}
}

// Do is the singleflight entrypoint. Calls fn at most once per
// outstanding key; concurrent callers with the same key share the
// result. Within sticky window after completion, the cached result is
// returned without calling fn.
func (s *sfGroup) Do(key string, fn func() (any, error)) (any, bool, error) {
	if s == nil {
		v, err := fn()
		return v, false, err
	}
	if key == "" {
		v, err := fn()
		return v, false, err
	}

	// Sticky-window check.
	s.mu.Lock()
	if p, ok := s.pending[key]; ok {
		if time.Now().Before(p.expiry) {
			value, err := p.value, p.err
			s.mu.Unlock()
			return value, true, err
		}
		delete(s.pending, key)
	}
	s.mu.Unlock()

	v, err, shared := s.g.Do(key, func() (any, error) {
		val, ferr := fn()
		s.mu.Lock()
		s.pending[key] = &sfPending{
			value:  val,
			err:    ferr,
			expiry: time.Now().Add(s.window),
		}
		s.mu.Unlock()
		return val, ferr
	})
	return v, shared, err
}

// Forget evicts the sticky entry for key. Safe on nil receiver.
func (s *sfGroup) Forget(key string) {
	if s == nil || key == "" {
		return
	}
	s.g.Forget(key)
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.pending, key)
}
