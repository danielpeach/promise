package promise

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// A Promise represents the eventual result of an operation.
// Retrieve a Promise's value using Await or Map.
type Promise[T any] struct {
	f         func(ctx context.Context) (T, error)
	value     T
	err       error
	resolved  bool
	rejected  bool
	cancel    context.CancelFunc
	ctx       context.Context
	callbacks []func(ctx context.Context, value T, err error)
	mu        *sync.Mutex
}

// New returns a new Promise. The provided function f will be executed immediately.
// New does not block on f's completion. The returned Promise represents the eventual return
// value of f.
func New[T any](ctx context.Context, f func(ctx context.Context) (T, error)) *Promise[T] {
	ctx, cancel := context.WithCancel(ctx)
	p := &Promise[T]{
		mu:     &sync.Mutex{},
		f:      f,
		ctx:    ctx,
		cancel: cancel,
	}

	launch(p)
	return p
}

// Resolve returns a Promise that has already been resolved. You will likely only need Resolve
// if you are working with (or creating) an API boundary that accepts or returns a Promise
// but you don't need to actually calculate a value.
func Resolve[T any](value T) *Promise[T] {
	ctx, cancel := context.WithCancel(context.Background())
	return &Promise[T]{
		resolved: true,
		value:    value,
		mu:       &sync.Mutex{},
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Reject returns a Promise that has already been rejected. You will likely only need Reject
// if you are working with (or creating) an API boundary that accepts or returns a Promise.
func Reject[T any](err error) *Promise[T] {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return &Promise[T]{
		rejected: true,
		err:      err,
		mu:       &sync.Mutex{},
		ctx:      ctx,
		cancel:   cancel,
	}
}

func launch[T any](p *Promise[T]) {
	go func(ctx context.Context) {
		doneChan := make(chan T)
		errChan := make(chan error)

		go func(ctx context.Context) {
			result, err := p.f(ctx)
			if err != nil {
				errChan <- err
				return
			}
			doneChan <- result
		}(ctx)

		select {
		case <-ctx.Done():
			p.mu.Lock()
			if err := ctx.Err(); err != nil {
				p.err = err
			} else {
				p.err = fmt.Errorf("context canceled")
			}
			p.rejected = true
			p.mu.Unlock()
		case err := <-errChan:
			p.mu.Lock()
			p.err = err
			p.rejected = true
			p.mu.Unlock()
		case result := <-doneChan:
			p.mu.Lock()
			p.value = result
			p.resolved = true
			p.mu.Unlock()
		}

		p.executeCallbacks(ctx)
	}(p.ctx)
}

func (p *Promise[T]) executeCallbacks(ctx context.Context) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, cb := range p.callbacks {
		go cb(ctx, p.value, p.err)
	}
}

// Await blocks until a Promise has been settled (i.e., either resolved or rejected) and returns
// its value (or error).
func (p *Promise[T]) Await() (T, error) {
	p.mu.Lock()
	if p.resolved || p.rejected {
		defer p.mu.Unlock()
		return p.value, p.err
	}

	doneChan := make(chan T)
	errChan := make(chan error)

	p.callbacks = append(p.callbacks, func(ctx context.Context, value T, err error) {
		if err != nil {
			errChan <- err
			return
		}
		doneChan <- value
	})
	p.mu.Unlock()

	select {
	case result := <-doneChan:
		return result, nil
	case err := <-errChan:
		return zeroValue[T](), err
	}
}

// Map returns a Promise by converting
func Map[T, U any](promise *Promise[T], f func(ctx context.Context, value T, err error) (U, error)) *Promise[U] {
	return New[U](promise.ctx, func(ctx context.Context) (U, error) {
		value, err := promise.Await()
		return f(ctx, value, err)
	})
}

// Then is an alias for Map.
func Then[T, U any](promise *Promise[T], f func(ctx context.Context, value T, err error) (U, error)) *Promise[U] {
	return Map[T, U](promise, f)
}

type promiseComplete[T any] struct {
	value T
	index int
}

// All returns a Promise that is fulfilled when all of the input promises have been fulfilled. Its
// value resolves to a slice of the results of the input promises.
func All[T any](ctx context.Context, promises ...*Promise[T]) *Promise[[]T] {
	return New(ctx, func(ctx context.Context) ([]T, error) {
		var results []promiseComplete[T]

		errChan := make(chan error)
		doneChan := make(chan promiseComplete[T])

		for index, promise := range promises {
			go func(i int, p *Promise[T]) {
				value, err := p.Await()
				if err != nil {
					errChan <- err
					return
				}
				doneChan <- promiseComplete[T]{
					value: value,
					index: i,
				}
			}(index, promise)
		}

		for len(results) != len(promises) {
			select {
			case err := <-errChan:
				return nil, err
			case result := <-doneChan:
				results = append(results, result)
			case <-ctx.Done():
				if err := ctx.Err(); err != nil {
					return nil, err
				} else {
					return nil, fmt.Errorf("context canceled")
				}
			}
		}

		sort.SliceStable(results, func(i, j int) bool {
			return results[i].index < results[j].index
		})

		return mapWith[promiseComplete[T], T](results, func(result promiseComplete[T]) T {
			return result.value
		}), nil
	})
}

// Race returns a Promise that is fulfilled or rejected as soon as one of the
// input promises is fulfilled or rejected, with the value or rejection reason from that promise.
func Race[T any](ctx context.Context, promises ...*Promise[T]) *Promise[T] {
	return New(ctx, func(ctx context.Context) (T, error) {
		errChan := make(chan error)
		doneChan := make(chan T)

		defer func() {
			for _, promise := range promises {
				promise.mu.Lock()
				if promise.cancel != nil {
					promise.cancel()
				}
				promise.mu.Unlock()
			}
		}()

		for _, promise := range promises {
			go func(p *Promise[T]) {
				value, err := p.Await()
				if err != nil {
					errChan <- err
					return
				}
				doneChan <- value
			}(promise)
		}

		select {
		case err := <-errChan:
			return zeroValue[T](), err
		case result := <-doneChan:
			return result, nil
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				return zeroValue[T](), err
			} else {
				return zeroValue[T](), fmt.Errorf("context canceled")
			}
		}
	})
}

type AggregateError struct {
	Errors []error
}

func (a *AggregateError) Error() string {
	return fmt.Sprintf("all promises failed: %s", strings.Join(mapWith[error, string](a.Errors, func(err error) string {
		return err.Error()
	}), ", "))
}

type promiseError struct {
	err   error
	index int
}

// Any returns a Promise that resolves as soon as any of the input promises is fulfilled, with
// the value of the fulfilled promise. If all of the input promises are rejected, then the
// returned promise is rejected with an AggregateError, which aggregates the errors from
// the input promises.
func Any[T any](ctx context.Context, promises ...*Promise[T]) *Promise[T] {
	return New(ctx, func(ctx context.Context) (T, error) {
		var errs []promiseError

		errChan := make(chan promiseError)
		doneChan := make(chan T)

		defer func() {
			for _, promise := range promises {
				promise.mu.Lock()
				if promise.cancel != nil {
					promise.cancel()
				}
				promise.mu.Unlock()
			}
		}()

		for index, promise := range promises {
			go func(i int, p *Promise[T]) {
				value, err := p.Await()
				if err != nil {
					errChan <- promiseError{
						err:   err,
						index: i,
					}
					return
				}
				doneChan <- value
			}(index, promise)
		}

		for len(errs) != len(promises) {
			select {
			case <-ctx.Done():
				if err := ctx.Err(); err != nil {
					return zeroValue[T](), err
				} else {
					return zeroValue[T](), fmt.Errorf("context canceled")
				}
			case err := <-errChan:
				errs = append(errs, err)
			case result := <-doneChan:
				return result, nil
			}
		}

		sort.SliceStable(errs, func(i, j int) bool {
			return errs[i].index < errs[j].index
		})

		return zeroValue[T](), &AggregateError{
			Errors: mapWith[promiseError, error](errs, func(pErr promiseError) error {
				return pErr.err
			}),
		}
	})
}

func mapWith[T, U any](in []T, f func(t T) U) (out []U) {
	for _, t := range in {
		out = append(out, f(t))
	}
	return
}

func zeroValue[T any]() T {
	return struct { t T } {}.t
}
