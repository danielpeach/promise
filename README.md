# Go Promises
Go Promises is inspired by JavaScript's [`Promise` implementation](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Promise).

## Usage

### `New[T any](ctx context.Context, f func(context.Context) (T, error))`

`New` returns a new `Promise`. Function `f` will be executed immediately. `New` 
does not block on `f`'s completion.

A `Promise` can be `pending`, `rejected`, or `resolved`:
1. While `f` is executing, the `Promise` is `pending`.
2. If `f` returns an `error`, the `Promise` is `rejected`. 
3. If `f` does not return an error, the `Promise` is `resolved` with the returned value of type `T`.

If the `ctx` passed to `New` is canceled, then the `Promise` will be rejected immediately. The `ctx` passed
to `f` will also be cancelled; `f` should pass `ctx` to subroutines and check if `ctx` has been cancelled, if applicable.

### `Promise[T any].Await() (T, error)`

`Await` blocks on the completion of a `Promise` and returns the resolved value of type `T` or rejection error.

You can combine `promise.New(...)` and `Await` to make IO-bound calls in parallel and then
block on their completion:

```go
promiseForSnakes := promise.New[[]snake](ctx, func(ctx context.Context) ([]snake, error) {
	// Fetch some snakes!
})

promiseForBees := promise.New[[]bee](ctx, func(ctx context.Context) ([]bee, error) {
	// Fetch some bees! 
})

snakes, err := promiseForSnakes.Await()
if err != nil { 
	// Handle snake fetching error (e.g., teeth too sharp!)
}

bees, err := promiseForBees.Await()
if err != nil { 
	// Handle bee fetching error (e.g., stingers too pointy!)
}

// Do something (e.g., fight!).
```

### `All[T any](ctx context.Context, promise... *Promise[T]) *Promise[[]T]`

`All` returns a `Promise` that resolves when all of the input promises have been resolved. Its 
value resolves to a slice of the results of the input promises.

```go
hats, err := promise.All[hat](
	ctx, 
	promise.New[hat](ctx, func(ctx context.Context) (hat, error) {
		// Fetch ten-gallon hat.
	}), 
	promise.New[hat](ctx, func(ctx context.Context) (hat, error) { 
		// Fetch fedora.
	}), 
	promise.New[hat](ctx, func(ctx context.Context) (hat, error) { 
		// Fetch balaclava.
	}),
).Await()
if err != nil { 
	// Handle hat error (e.g., hat only nine-gallons!)
}
for _, hat := range hats {
	// Put on hat.
}

```

### `Race(ctx context.Context, promise... *Promise[T]) *Promise[T]`

`Race` returns a `Promise` that is resolved or rejected as soon as one of the
input promises is resolved or rejected, with the value or rejection error from that promise.

```go
winner, err := promise.Race[racer](
	ctx, 
	promise.New[racer](ctx, func(ctx context.Context) (racer, error) { 
		// Launch hare. 
	}), 
	promise.New[hat](ctx, func(ctx context.Context) (racer, error) { 
		// Launch tortoise.
	}),
).Await()
if err != nil { 
	// Handle race error (e.g., hare had sunstroke).
}
// Congratulate tortoise.
```

### `Any(ctx context.Context, promise... *Promise[T]) *Promise[T]`

`Any` returns a `Promise` that resolves as soon as any of the input promises is resolved, with
the value of the resolved promise. If all of the input promises are rejected, then the
returned promise is rejected with an `AggregateError`, which aggregates the errors from
the input promises.

```go
survivor, err := promise.Any[catFighter](
    ctx,
    promise.New[catFighter](ctx, func(ctx context.Context) (catFighter, error) {
        // Fight lion.
    }),
    promise.New[catFighter](ctx, func(ctx context.Context) (catFighter, error) {
        // Fight snow leopard.
    }),
    promise.New[catFighter](ctx, func(ctx context.Context) (catFighter, error) {
        // Fight jaguar. 
    }),
).Await()
if err != nil { 
	// Handle error (e.g., all cats are victors).
}
// Congratulate survivor.
```
