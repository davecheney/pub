// package group provides a way to manage the lifecycle of a group of goroutines.
package group

import (
	"context"
	"sync"
)

// A G manages the lifetime of a set of goroutines from a common context.
// The first goroutine in the group to return will cause the context to be canceled,
// terminating the remaining goroutines.
type G struct {
	// ctx is the context passed to all goroutines in the group.
	ctx    context.Context
	cancel context.CancelFunc
	done   sync.WaitGroup

	errOnce sync.Once
	err     error
}

// newGroup returns a new group using the given context.
func New(ctx context.Context) *G {
	ctx, cancel := context.WithCancel(ctx)
	return &G{
		ctx:    ctx,
		cancel: cancel,
	}
}

// AddContext adds a new goroutine to the group.
// The goroutine should exit when the context passed to it is canceled.
func (g *G) AddContext(fn func(context.Context) error) {
	g.done.Add(1)
	go func() {
		defer g.done.Done()
		defer g.cancel()
		if err := fn(g.ctx); err != nil {
			g.errOnce.Do(func() { g.err = err })
		}
	}()
}

// Add adds a new goroutine to the group.
// The goroutine should exit when the channel passed to it is canceled.
func (g *G) Add(fn func(<-chan struct{}) error) {
	g.done.Add(1)
	go func() {
		defer g.done.Done()
		defer g.cancel()
		if err := fn(g.ctx.Done()); err != nil {
			g.errOnce.Do(func() { g.err = err })
		}
	}()
}

// Wait waits for all goroutines in the group to exit.
// If any of the goroutines fail with an error, Wait will return the first error.
func (g *G) Wait() error {
	g.done.Wait()
	g.errOnce.Do(func() {
		// noop, required to synchronise on the errOnce mutex.
	})
	return g.err
}
